import { useState, useCallback, useEffect } from 'react';

export interface ToastMessage {
  id: string;
  type: 'success' | 'error';
  text: string;
}

let toastListeners = new Set<(toasts: ToastMessage[]) => void>();
let toasts: ToastMessage[] = [];

const notifyListeners = () => {
  toastListeners.forEach((listener) => listener([...toasts]));
};

export const showToast = (type: 'success' | 'error', text: string): void => {
  const id = Date.now().toString();
  toasts.push({ id, type, text });
  notifyListeners();

  setTimeout(() => {
    toasts = toasts.filter((t) => t.id !== id);
    notifyListeners();
  }, 3000);
};

export const useToast = () => {
  const [currentToasts, setCurrentToasts] = useState<ToastMessage[]>([]);

  useEffect(() => {
    toastListeners.add(setCurrentToasts);
    return () => {
      toastListeners.delete(setCurrentToasts);
    };
  }, []);

  const showSuccess = useCallback((text: string) => {
    showToast('success', text);
  }, []);

  const showError = useCallback((text: string) => {
    showToast('error', text);
  }, []);

  return {
    toasts: currentToasts,
    showSuccess,
    showError,
    removeToast: (id: string) => {
      toasts = toasts.filter((t) => t.id !== id);
      notifyListeners();
    },
  };
};
