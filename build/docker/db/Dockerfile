FROM postgres:15
RUN localedef -i ru_RU -c -f UTF-8 -A /usr/share/locale/locale.alias ru_RU.UTF-8
ENV LANG ru_RU.UTF8

COPY init.sql /docker-entrypoint-initdb.d/10-init.sql