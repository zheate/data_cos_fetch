import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { ThemeProvider } from 'next-themes';
import './index.css';
import App from './App.tsx';
import { Toaster } from '@/components/ui/sonner';

import React from 'react';

class ErrorBoundary extends React.Component<{children: React.ReactNode}, {hasError: boolean, error: unknown}> {
  constructor(props: {children: React.ReactNode}) {
    super(props);
    this.state = { hasError: false, error: null };
  }
  static getDerivedStateFromError(error: unknown) { return { hasError: true, error }; }
  render() {
    if (this.state.hasError) return <div style={{padding: 20, color: 'red', background: '#fff', fontSize: 16}}><h2>React Error</h2><pre>{this.state.error instanceof Error ? this.state.error.stack : String(this.state.error)}</pre></div>;
    return this.props.children;
  }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem={false}>
      <ErrorBoundary>
        <App />
        <Toaster position="top-right" />
      </ErrorBoundary>
    </ThemeProvider>
  </StrictMode>,
);
