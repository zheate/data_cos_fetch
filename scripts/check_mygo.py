#!/usr/bin/env python3
"""Exercise only the task-owned MyGo app through its test-only automation socket."""
from __future__ import annotations

import argparse
import base64
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = Path(os.environ.get("DATA_COS_UI_OUTPUT", ROOT / "output/playwright/uix-refactor"))


class Client:
    def __init__(self, path: str):
        self.socket = socket.socket(socket.AF_UNIX)
        self.socket.settimeout(15)
        self.socket.connect(path)
        self.file = self.socket.makefile("rw", encoding="utf-8")
        self.counter = 0

    def call(self, method: str, **params):
        self.counter += 1
        self.file.write(json.dumps({"id": self.counter, "method": method, "params": params}) + "\n")
        self.file.flush()
        result = json.loads(self.file.readline())
        if "error" in result:
            raise RuntimeError(f"{method}: {result['error']}")
        return result["result"]

    def nodes(self):
        return self.call("tree")["nodes"]

    def node(self, label: str, role=None):
        return next(n for n in self.nodes() if n.get("label") == label and (role is None or n.get("role") == role))

    def act(self, *, key=None, label=None, role=None, action="press", text=None):
        params = {"action": action}
        if key is not None:
            params["key"] = key
        else:
            params["id"] = self.node(label, role)["id"]
        if text is not None:
            params["text"] = text
        self.call("act", **params)
        self.call("idle", timeout=5000)

    def wait(self, predicate, timeout=10):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            try:
                nodes = self.nodes()
            except RuntimeError as error:
                if "no_window" not in str(error):
                    raise
                time.sleep(.05)
                continue
            if predicate(nodes):
                return nodes
            time.sleep(.05)
        raise AssertionError("UI condition timed out")

    def capture(self, name):
        image = self.call("capture")
        (OUTPUT / name).write_bytes(base64.b64decode(image["png"]))
        return {"file": name, "width": image["width"], "height": image["height"]}

    def close(self):
        self.file.close()
        self.socket.close()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--engine", required=True)
    args = parser.parse_args()
    OUTPUT.mkdir(parents=True, exist_ok=True)
    results = []
    captures = []
    with tempfile.TemporaryDirectory(prefix="data-cos-e2e-", dir="/tmp") as temp:
        work = Path(temp)
        module_root = work / "modules"
        ids = ["FCP26111301", "FCP26111302", "FCP26111303", "FCP26111304"]
        fixture = ROOT / ".tmp/mygo-fixtures/module-01/耦合测试/测试"
        for entry in ids:
            dest = module_root.joinpath(*entry) / "耦合测试/测试"
            dest.mkdir(parents=True)
            for src in fixture.glob("*.xlsx"):
                shutil.copy2(src, dest / src.name)
        settings = work / "settings.json"
        settings.write_text(json.dumps({"mode": 0, "moduleRoot": str(module_root), "measurements": [True, True, False], "tests": [True, False, False, False, False, False], "currents": "a"}))
        sock = str(work / "app.sock")
        env = {**os.environ, "MYGO_AUTOMATION": sock, "DATA_COS_SETTINGS_PATH": str(settings), "DATA_COS_EXTRACT_BIN": args.engine}
        client = None
        with (OUTPUT / "native-e2e.log").open("w") as log:
            process = subprocess.Popen([args.binary], cwd=ROOT / "native", env=env, stdout=log, stderr=log)
            try:
                deadline = time.monotonic() + 20
                while not Path(sock).exists():
                    if process.poll() is not None:
                        raise RuntimeError("MyGo exited before opening its automation socket")
                    if time.monotonic() > deadline:
                        raise TimeoutError("MyGo socket startup timeout")
                    time.sleep(.05)
                client = Client(sock)
                client.wait(lambda nodes: any(n.get("key") == "entries" for n in nodes))
                assert "disabled" in client.node("开始提取").get("states", [])
                results.append("empty input disables extraction")
                # A genuine window with no webview, HTML or JavaScript frontend.
                window = client.call("windows")["windows"][0]
                assert window["kind"] == "content", window
                client.act(key="entries", action="setValue", text="\n".join(ids + [ids[0]]))
                client.act(key="tidy-entries")
                assert client.node("编号或完整路径清单")["value"] == "\n".join(ids)
                results.append("entry cleanup preserves order and removes duplicates")
                client.act(key="currents", action="setValue", text="2oops")
                assert "invalid" in client.node("电流点 / A").get("states", [])
                assert "disabled" in client.node("开始提取").get("states", [])
                results.append("invalid current shows inline feedback and disables extraction")
                client.act(label="最大值")
                assert not client.node("电流点 / A").get("value", "")
                client.act(label="全部")
                assert client.node("电流点 / A")["value"] == "a"
                results.append("maximum and all-current quick actions update the input")
                client.act(key="run")
                client.wait(lambda ns: any(n.get("label", "").startswith("提取完成 · 12 条记录") for n in ns))
                assert client.node("测试数据提取结果")
                assert any(n.get("label") == "显示 12 / 12 条" for n in client.nodes())
                results.append("module extraction: 4 identifiers, 12 merged LVI/Rth records")
                captures.append(client.capture("after-native-module.png"))
                cell = client.node(ids[0])
                bounds = cell["bounds"]
                client.call("input", events=[{"type":"click","x":bounds["x"]+bounds["width"]/2,"y":bounds["y"]+bounds["height"]/2}])
                client.call("idle", timeout=5000)
                assert any(n.get("label") == "选中 1 条" for n in client.nodes())
                assert client.node("复制选中记录")
                results.append("result row selection makes the copy scope explicit")
                client.call("input", events=[{"type":"key","key":"CmdOrCtrl+F"}])
                client.call("idle", timeout=5000)
                tree = client.call("tree")
                assert tree.get("focus") == client.node("搜索编号或站别")["id"]
                results.append("keyboard shortcut focuses result search")
                client.act(label="搜索编号或站别", action="setValue", text=ids[1])
                assert any(n.get("label") == "显示 3 / 12 条" for n in client.nodes())
                results.append("search filters the displayed records")
                client.act(label="搜索编号或站别", action="setValue", text="no-match")
                assert client.node("没有匹配的记录")
                client.act(label="清除搜索")
                results.append("empty search results provide a working reset")
                client.act(key="settings")
                assert any(n.get("key") == "module-root" for n in client.nodes())
                captures.append(client.capture("after-native-settings-dialog.png"))
                client.act(label="完成")
                assert not any(n.get("key") == "module-root" for n in client.nodes())
                results.append("secondary directory settings stay outside the main workflow")
                client.act(key="custom-button")
                client.act(key="custom-tests", action="setValue", text="自定义测试站别")
                client.call("input", events=[{"type": "key", "key": "Escape"}])
                client.call("idle", timeout=5000)
                assert not any(n.get("role") == "dialog" for n in client.nodes())
                client.act(key="custom-button")
                assert not client.node("自定义测试站别清单").get("value", "")
                captures.append(client.capture("after-native-cancel-dialog.png"))
                client.act(label="取消")
                results.append("Chinese draft input is discarded by Escape and cancel")
                client.act(key="custom-button")
                client.act(key="custom-tests", action="setValue", text="老化后测试\n温循后测试\n老化后测试")
                client.act(label="保存站别")
                assert client.node("老化后测试") and client.node("温循后测试")
                assert client.node("3 个已选")
                assert client.node("自定义测试站别")
                captures.append(client.capture("after-native-custom-stations.png"))
                # Custom station names remain part of the real Rust extraction request.
                for entry in ids:
                    source = module_root.joinpath(*entry) / "耦合测试/测试"
                    for name in ["老化后测试", "温循后测试"]:
                        shutil.copytree(source, source.parent.parent / name / "测试")
                client.act(key="run")
                client.wait(lambda ns: any(n.get("label", "").startswith("提取完成 · 36 条记录") for n in ns))
                results.append("saved Chinese custom stations appear immediately and produce 36 records across three stations")
                client.act(label="老化后测试", role="checkBox")
                assert client.node("2 个已选")
                client.act(key="run")
                client.wait(lambda ns: any(n.get("label", "").startswith("提取完成 · 24 条记录") for n in ns))
                results.append("unchecking a custom station excludes it from the real extraction request")
                client.act(label="全选测试站别")
                assert client.node("8 个已选")
                client.act(label="清空测试站别")
                assert client.node("0 个已选")
                assert "disabled" in client.node("开始提取").get("states", [])
                client.act(label="耦合测试", role="checkBox")
                results.append("select all and clear apply equally to built-in and custom stations")
                client.act(label="温循后测试", role="checkBox")
                assert client.node("2 个已选")
                client.act(key="delete-tests")
                assert client.node("0 个已选")
                assert "disabled" in client.node("删除选中站别").get("states", [])
                assert not any(n.get("role") == "checkBox" and n.get("label") in ["耦合测试", "温循后测试"] for n in client.nodes())
                assert client.node("老化后测试", role="checkBox")
                client.act(label="老化后测试", role="checkBox")
                client.act(key="run")
                client.wait(lambda ns: any(n.get("label", "").startswith("提取完成 · 12 条记录") for n in ns))
                captures.append(client.capture("after-native-station-delete.png"))
                results.append("bulk deletion removes only checked built-in and custom stations; the remaining station extracts 12 records")
                client.act(key="custom-button")
                client.act(key="custom-tests", action="setValue", text="")
                client.act(label="保存站别")
                assert not any(n.get("label") in ["老化后测试", "温循后测试"] and n.get("bounds", {}).get("x", 1000) < 360 for n in client.nodes())
                assert client.node("0 个已选")
                results.append("clearing custom stations updates the station list and count")
                client.act(label="芯片")
                client.act(key="entries", action="setValue", text=str(ROOT / ".tmp/mygo-fixtures/chip-01"))
                client.act(key="run")
                client.wait(lambda ns: any(n.get("label", "").startswith("提取完成 · 3 条记录") for n in ns))
                results.append("chip extraction: postQCW, 3 merged records")
                client.act(key="entries", action="setValue", text=str(work / "missing-chip"))
                client.act(key="run")
                client.wait(lambda ns: any(n.get("label", "").startswith("部分完成 · 0 条记录") for n in ns))
                assert any("chip folder not found" in n.get("label", "") for n in client.nodes())
                results.append("missing source produces a visible diagnostic")
                client.act(key="clear-results")
                assert client.node("从一组编号开始")
                results.append("clear result restores the initial state")
            finally:
                if client is not None:
                    client.close()
                if process.poll() is None:
                    process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
    report = {"checks": results, "captures": captures, "passed": True}
    (OUTPUT / "native-e2e.json").write_text(json.dumps(report, ensure_ascii=False, indent=2))
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
