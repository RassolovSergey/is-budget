import type { Metadata } from "next";
import { Noto_Sans, Noto_Serif } from "next/font/google";

import { ConfirmProvider } from "../components/ConfirmProvider";
import { LocaleProvider } from "../components/LocaleProvider";
import { ThemeProvider } from "../components/ThemeProvider";

import "./globals.css";

const sans = Noto_Sans({ subsets: ["latin", "cyrillic"], variable: "--font-grotesk" });
const serif = Noto_Serif({ subsets: ["latin", "cyrillic"], variable: "--font-fraunces" });

export const metadata: Metadata = {
  title: "IS Budget",
  description: "Прототип системы планирования расходов, прогнозирования и распределения прибыли"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" suppressHydrationWarning>
      <body className={`${sans.variable} ${serif.variable}`}>
        <LocaleProvider>
          <ThemeProvider>
            <ConfirmProvider>{children}</ConfirmProvider>
          </ThemeProvider>
        </LocaleProvider>
      </body>
    </html>
  );
}
