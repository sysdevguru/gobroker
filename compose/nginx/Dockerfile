FROM nginx:1.13.0-alpine

RUN rm -rf /etc/nginx/conf.d
COPY conf.d/ /etc/nginx/conf.d/

EXPOSE 80 443
CMD ["nginx", "-g", "daemon off;"]
