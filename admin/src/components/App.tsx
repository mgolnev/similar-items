import React, { Suspense } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { SparklesIcon, Squares2X2Icon, CloudArrowUpIcon, SquaresPlusIcon } from '@heroicons/react/24/outline';
import { SimilarShowcase } from '../pages/SimilarShowcase';
import { Feed } from '../pages/Feed';

const SimilarStrategy = React.lazy(() =>
  import('../pages/SimilarStrategy').then((module) => ({ default: module.SimilarStrategy }))
);

const OutfitStrategy = React.lazy(() =>
  import('../pages/OutfitStrategy').then((module) => ({ default: module.OutfitStrategy }))
);

const Navigation: React.FC = () => {
  const location = useLocation();

  const navItems = [
    { path: '/similar-strategy', label: 'Похожие — настройки', icon: SparklesIcon },
    { path: '/outfit-strategy', label: 'Комплекты', icon: SquaresPlusIcon },
    { path: '/showcase', label: 'Витрина', icon: Squares2X2Icon },
    { path: '/feed', label: 'Фид', icon: CloudArrowUpIcon },
  ];

  return (
    <nav className="relative z-20 bg-white shadow-sm border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-12">
          <div className="flex">
            <div className="flex-shrink-0 flex items-center">
              <h1 className="text-lg font-bold text-gray-900">GJ Recommendations</h1>
            </div>
            <div className="hidden sm:ml-6 sm:flex sm:space-x-6">
              {navItems.map((item) => {
                const Icon = item.icon;
                const isActive = location.pathname.startsWith(item.path);
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={`inline-flex items-center px-1 pt-1 border-b-2 text-xs font-medium transition-colors duration-200 ${
                      isActive
                        ? 'border-primary-500 text-primary-600'
                        : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                    }`}
                  >
                    <Icon className="w-4 h-4 mr-1.5" />
                    {item.label}
                  </Link>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
};

export const App: React.FC = () => {
  return (
    <Router>
      <div className="min-h-screen bg-gray-50">
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              background: '#fff',
              color: '#374151',
              boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
              border: '1px solid #e5e7eb',
              borderRadius: '0.75rem',
              padding: '16px',
            },
            success: { iconTheme: { primary: '#10b981', secondary: '#fff' } },
            error: { iconTheme: { primary: '#ef4444', secondary: '#fff' } },
          }}
        />

        <Navigation />

        <Routes>
          <Route path="/" element={<Navigate to="/similar-strategy" replace />} />
          <Route
            path="/similar-strategy"
            element={
              <Suspense fallback={<div className="p-6 text-sm text-gray-500">Загрузка настроек похожих...</div>}>
                <SimilarStrategy />
              </Suspense>
            }
          />
          <Route
            path="/outfit-strategy"
            element={
              <Suspense fallback={<div className="p-6 text-sm text-gray-500">Загрузка настроек комплектов...</div>}>
                <OutfitStrategy />
              </Suspense>
            }
          />
          <Route path="/showcase" element={<SimilarShowcase />} />
          <Route path="/feed" element={<Feed />} />
        </Routes>
      </div>
    </Router>
  );
};
