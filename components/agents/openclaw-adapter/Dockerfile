FROM node:24-alpine
WORKDIR /app
RUN npm install yaml
COPY seed-config.mjs test.sh ./
ENTRYPOINT ["node", "seed-config.mjs"]
