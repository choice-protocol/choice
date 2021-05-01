FROM nginx

ENV JSONRPC_URL=https://mainnet.infura.io/v3/c5b349fd47244da8a4df10652b911d38

COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
