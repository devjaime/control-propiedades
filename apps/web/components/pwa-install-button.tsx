"use client";

import { useEffect, useState } from "react";

type InstallPromptEvent = Event & {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
};

export function PwaInstallButton() {
  const [installPrompt, setInstallPrompt] = useState<InstallPromptEvent | null>(null);

  useEffect(() => {
    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.register("/sw.js").catch(() => undefined);
    }

    function captureInstallPrompt(event: Event) {
      event.preventDefault();
      setInstallPrompt(event as InstallPromptEvent);
    }

    function clearInstallPrompt() {
      setInstallPrompt(null);
    }

    window.addEventListener("beforeinstallprompt", captureInstallPrompt);
    window.addEventListener("appinstalled", clearInstallPrompt);
    return () => {
      window.removeEventListener("beforeinstallprompt", captureInstallPrompt);
      window.removeEventListener("appinstalled", clearInstallPrompt);
    };
  }, []);

  if (!installPrompt) return null;

  async function install() {
    if (!installPrompt) return;
    await installPrompt.prompt();
    await installPrompt.userChoice;
    setInstallPrompt(null);
  }

  return (
    <button className="install-button" onClick={install} type="button">
      Instalar app
    </button>
  );
}
