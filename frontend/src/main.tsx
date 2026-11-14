import React from "react";
import ReactDOM from "react-dom/client";
import { createBrowserRouter, RouterProvider, Navigate } from "react-router-dom";
import "./index.css";
import { SettingsProvider } from "./ui/settings";
import { applyFavicon } from "./brand";
import { Layout } from "./ui/Layout";
import Dashboard from "./pages/Dashboard";
import Leads from "./pages/Leads";
import LeadDetail from "./pages/LeadDetail";
import Units from "./pages/Units";
import Deals from "./pages/Deals";
import DealDetail from "./pages/DealDetail";
import Agents from "./pages/Agents";
import Agencies from "./pages/Agencies";
import Reports from "./pages/Reports";
import ReportPrint from "./pages/ReportPrint";
import Search from "./pages/Search";
import Commissions from "./pages/Commissions";
import CommissionsPrint from "./pages/CommissionsPrint";

const router = createBrowserRouter([
  {
    element: <Layout />,
    children: [
      { path: "/", element: <Navigate to="/panel" replace /> },
      { path: "/panel", element: <Dashboard /> },
      { path: "/musteriler", element: <Leads /> },
      { path: "/musteriler/:id", element: <LeadDetail /> },
      { path: "/daireler", element: <Units /> },
      { path: "/satislar", element: <Deals /> },
      { path: "/satislar/:id", element: <DealDetail /> },
      { path: "/raporlar", element: <Reports /> },
      { path: "/personel", element: <Agents /> },
      { path: "/acentalar", element: <Agencies /> },
      { path: "/komisyonlar", element: <Commissions /> },
      { path: "/ara", element: <Search /> },
    ],
  },
  // the printable report lives outside the app shell: no sidebar, no top bar, A4 only
  { path: "/raporlar/pdf", element: <ReportPrint /> },
  { path: "/komisyonlar/pdf", element: <CommissionsPrint /> },
]);

applyFavicon();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <SettingsProvider>
      <RouterProvider router={router} />
    </SettingsProvider>
  </React.StrictMode>,
);
