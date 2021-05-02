const http = require('http');
const httpProxy = require('http-proxy');
const express = require('express');
const bodyParser = require('body-parser');

const serverPort = 8545;
const proxyTarget = 'https://mainnet.infura.io/v3/c5b349fd47244da8a4df10652b911d38';

function logRequest(request) {
  const logObj = {};

  logObj.url = request.url;
  logObj.method = request.method;
  logObj.headers = request.headers;
  logObj.httpVersion = request.httpVersion;

  if (request.rawBody) {
    logObj.rawBody = request.rawBody;
  }

  if (request.body) {
    logObj.body = request.body;
  }

  console.log(JSON.stringify(logObj, true, 2));
  // console.log(request);
}
/////////////////////////////////// PROXY SERVER ///////////////////////////////

const proxy = httpProxy.createProxyServer({});

// Handle Error
proxy.on('error', function (err, req, res) {
  res.writeHead(500, {
    'Content-Type': 'text/plain',
  });
  res.end(`Something went wrong. \n\n\n${err}`);
});

// Restream parsed body before proxying
proxy.on('proxyReq', function (proxyReq, req, res, options) {
  logRequest(req);
});

const proxyApp = express();
proxyApp.use(bodyParser.json());
proxyApp.use(bodyParser.urlencoded({ extended: true }));
proxyApp.use(function (req, res) {
  // ... do some stuff
  // ... log your body and something else
  console.log('proxy body:', req.body);
  proxy.web(req, res, {
    target: 'http://127.0.0.1:5000',
  });
});

http.createServer(proxyApp).listen(serverPort, '0.0.0.0', () => {
  console.log(`Proxy server linsten on ${serverPort}`);
});
