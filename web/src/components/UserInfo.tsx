'use client';

import { useAuth } from '../hooks/useAuth';
import { Button } from './ui/button';

export function UserInfo() {
  const { user, isAuthenticated, logout } = useAuth();

  if (!isAuthenticated || !user) return null;

  return (
    <div className="absolute top-4 right-4 z-[1000] bg-white rounded-lg shadow-md p-3 flex items-center gap-3">
      <div className="text-sm">
        <p className="font-medium text-gray-900">{user.name}</p>
        <p className="text-gray-500 text-xs">{user.email}</p>
      </div>
      <Button variant="outline" size="sm" onClick={logout}>
        Logout
      </Button>
    </div>
  );
}
