FROM postgis/postgis:16-3.4@sha256:44126d872ac91993766c341e369c539e8196614321765d36a6f1bab0419a5fa5

RUN sed -i 's#http://apt.postgresql.org/pub/repos/apt/#https://apt-archive.postgresql.org/pub/repos/apt/#' /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends postgresql-16-pgvector \
    && rm -rf /var/lib/apt/lists/*

COPY init.sql /docker-entrypoint-initdb.d/01-extensions.sql
