// @vitest-environment jsdom
import React from 'react';
import { act, cleanup, render } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { ToastProvider, useToast } from '../src/context/ToastContext';
let toast: ReturnType<typeof useToast>;
function Probe() { toast = useToast(); return null; }
afterEach(() => { cleanup(); vi.useRealTimers(); });
it('clears toast timers and settles pending confirmation on unmount', async () => {
  vi.useFakeTimers();
  const view = render(<ToastProvider><Probe /></ToastProvider>);
  let pending!: Promise<boolean>;
  act(() => { toast.addToast('test'); pending = toast.confirm('Continue?'); });
  expect(vi.getTimerCount()).toBeGreaterThan(0);
  view.unmount();
  expect(vi.getTimerCount()).toBe(0);
  await expect(pending).resolves.toBe(false);
});
it('settles the previous confirmation when another replaces it', async () => {
  render(<ToastProvider><Probe /></ToastProvider>);
  let first!: Promise<boolean>;
  act(() => { first = toast.confirm('First?'); });
  act(() => { void toast.confirm('Second?'); });
  await expect(first).resolves.toBe(false);
});
