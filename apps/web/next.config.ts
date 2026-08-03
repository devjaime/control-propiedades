import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  experimental: {
    // 50 MB de archivo más el pequeño overhead del formulario multipart.
    proxyClientMaxBodySize: "52mb",
  },
  async rewrites() {
    const apiURL = process.env.API_INTERNAL_URL ?? "http://localhost:8080";
    return [
      {
        source: "/api/:path*",
        destination: `${apiURL}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
