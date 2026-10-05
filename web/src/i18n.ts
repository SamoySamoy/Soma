import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import en from "./locales/en.json";

// English only for now (business spec decision, 2026-10-04). Every
// user-facing string lives in src/locales so more languages can be added.
void i18n.use(initReactI18next).init({
  lng: "en",
  fallbackLng: "en",
  resources: { en: { translation: en } },
  interpolation: { escapeValue: false }, // React already escapes output
});

export default i18n;
