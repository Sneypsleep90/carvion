import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { desktopBackend } from "./lib/backend";
import "./index.css";

const root = document.getElementById("root");
if (!root) {
  throw new Error("CARVION Studio root element is missing");
}

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <App backend={desktopBackend} />
  </React.StrictMode>,
);
