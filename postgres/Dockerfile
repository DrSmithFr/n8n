# This is installing the pgvector extension for postgres
FROM postgres:16

# Install requirements
RUN apt-get update && \
    apt-get install -y git build-essential postgresql-server-dev-16 && \
    git clone https://github.com/pgvector/pgvector.git && \
    cd pgvector && \
    make && \
    make install && \
    cd .. && \
    rm -rf pgvector && \
    apt-get remove --purge -y build-essential postgresql-server-dev-16 && \
    apt-get autoremove -y && \
    apt-get clean

# Add the extension to the PostgreSQL configuration
RUN echo "shared_preload_libraries = 'pgvector'" >> /usr/share/postgresql/postgresql.conf.sample

# Loading extension
COPY ./vector_extension.sql /docker-entrypoint-initdb.d/init_pgvector.sql