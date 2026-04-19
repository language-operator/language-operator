FROM node:24-alpine
WORKDIR /app
RUN apk add --no-cache git github-cli bash
COPY package.json ./
RUN npm install
COPY server.mjs test.sh ./
ENTRYPOINT ["node", "server.mjs"]
