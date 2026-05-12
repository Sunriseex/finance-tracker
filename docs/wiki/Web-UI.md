# Web UI

The Web UI lives in `web/`.

## Stack

* React
* Vite
* TypeScript
* TanStack Query
* Recharts

## Local Development

Start the API first, then run:

```bash
cd web
npm install
npm run dev
```

The Vite dev server proxies API requests. Check `web/` config for the current proxy target.

## Auth Client Behavior

The Web UI uses access tokens for API mutations and reads. Refresh/logout can use the refresh token JSON body and the secure refresh cookie fallback.

See [Auth Security Model](Auth-Security-Model).
