FROM node:22-alpine AS build

WORKDIR /app
COPY package.json yarn.lock .yarnrc.yml ./
RUN corepack enable \
  && corepack prepare yarn@4.9.2 --activate \
  && yarn install --immutable
COPY index.html tsconfig.json tsconfig.app.json tsconfig.node.json vite.config.ts ./
COPY public ./public
COPY src ./src
RUN yarn build

FROM nginx:1.29-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.d/40-txsim-config.sh /docker-entrypoint.d/40-txsim-config.sh
RUN chmod +x /docker-entrypoint.d/40-txsim-config.sh
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
