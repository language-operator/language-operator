import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "./providers";

// Use system fonts to avoid network dependency during build
const geistSans = {
  variable: "--font-geist-sans",
  className: "",
};

const geistMono = {
  variable: "--font-geist-mono", 
  className: "",
};

export const metadata: Metadata = {
  title: "Language Operator Dashboard",
  description: "Manage your Language Operator resources",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
