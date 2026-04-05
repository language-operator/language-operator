FROM node:24-alpine
WORKDIR /app
COPY package.json ./
RUN npm install
COPY server.mjs test.sh ./
ENTRYPOINT ["node", "server.mjs"]
