import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import UnitPage from './units/page';
import CategoryPage from './categories/page';
import PackingPage from './packings/page';
import PlantPage from './plants/page';

vi.mock('../components/app-shell/app-shell', () => ({
  AppShell: ({ children }: { children: React.ReactNode }) => <>{children}</>
}));

vi.mock('../components/master-data-crud', () => ({
  MasterDataCrud: ({ title, endpoint }: { title: string; endpoint: string }) => <div data-testid={endpoint}>{title}</div>
}));

describe('reference master pages', () => {
  it.each([
    [UnitPage, 'Units', '/master-data/units'],
    [CategoryPage, 'Categories', '/master-data/categories'],
    [PackingPage, 'Packings', '/master-data/packings'],
    [PlantPage, 'Plants', '/master-data/plants']
  ])('binds %s to its API endpoint', (Page, title, endpoint) => {
    render(<Page />);
    expect(screen.getByTestId(endpoint)).toHaveTextContent(title);
  });
});
