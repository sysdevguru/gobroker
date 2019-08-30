FROM fluent/fluentd:v1.0

USER root

RUN apk add --no-cache curl ruby-bigdecimal && \
    gem install fluent-plugin-s3 --no-ri --no-rdoc && \
    cd /tmp && curl -L https://github.com/progrium/entrykit/releases/download/v0.4.0/entrykit_0.4.0_Linux_x86_64.tgz > /tmp/entrykit.tgz && \
    tar zxvf /tmp/entrykit.tgz && mv /tmp/entrykit /bin/entrykit && /bin/entrykit --symlink && \
    apk del curl && rm -rf /tmp/* /var/tmp/* /var/cache/apk/*

COPY fluent.debug.conf /fluentd/etc/fluent.conf.tmpl

ENTRYPOINT [ \
  "render", "/fluentd/etc/fluent.conf", "--", \
  "switch", \
    "shell=sh", \
    "run=fluentd -c /fluentd/etc/fluent.conf -p /fluentd/plugins", \
    "debugrun=fluentd -c /fluentd/etc/fluent.conf -p /fluentd/plugins -vv", "--", \
  "sh"]
