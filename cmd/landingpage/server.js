import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import path from 'path';
import { fileURLToPath } from 'url';
import cors from 'cors';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const port = process.env.PORT || 3000;
const proxyAPI = process.env.PROXY_API_URL || 'http://localhost:8080';

app.use(cors());

// Proxy /api/apps to the MyProxy API
app.use(createProxyMiddleware({
    target: proxyAPI,
    changeOrigin: true,
    pathFilter: '/api/apps',
    onProxyRes: function (proxyRes, req, res) {
        proxyRes.headers['Access-Control-Allow-Origin'] = '*';
    },
    onError: (err, req, res) => {
        console.error('Proxy Error:', err);
        res.status(502).send('Proxy Error');
    }
}));

// Serve static files from the Vite build directory
app.use(express.static(path.join(__dirname, 'dist')));

// Fallback to index.html for SPA routing
app.use((req, res) => {
    res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(port, () => {
    console.log(`Landing page server running on port ${port}`);
    console.log(`Proxying /api/apps to ${proxyAPI}`);
});
