'use client';

import { useAppSelector, useAppDispatch } from '../store/store';
import { dismissError } from '../store/slices/uiSlice';
import { useEffect } from 'react';

export function GlobalErrorToast() {
  const errors = useAppSelector((s) => s.ui.errors);
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (errors.length === 0) return;

    const latest = errors[errors.length - 1];
    const timer = setTimeout(() => {
      dispatch(dismissError(latest.id));
    }, 5000);

    return () => clearTimeout(timer);
  }, [errors, dispatch]);

  if (errors.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-[99999] flex flex-col gap-2 max-w-sm">
      {errors.map((error) => (
        <div
          key={error.id}
          className="bg-red-50 border border-red-200 text-red-800 rounded-lg px-4 py-3 shadow-lg flex items-start gap-3 animate-in slide-in-from-right"
        >
          <div className="flex-1 text-sm">
            <p className="font-medium">
              {error.status ? `Error ${error.status}` : 'Error'}
            </p>
            <p className="mt-0.5 text-red-600">{error.message}</p>
          </div>
          <button
            onClick={() => dispatch(dismissError(error.id))}
            className="text-red-400 hover:text-red-600 text-lg leading-none"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
