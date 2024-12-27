# This is installing the pgvector extension for postgres
FROM postgres:16

# Install requirements
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    postgresql-server-dev-all \
    && rm -rf /var/lib/apt/lists/*

# Clone, build, and install the pgvector extension
RUN cd /tmp \
    && git clone --branch v0.5.0 https://github.com/pgvector/pgvector.git \
    && cd pgvector \
    && make \
    && make install

# Loading extension
COPY ./vector_extension.sql /docker-entrypoint-initdb.d/init_pgvector.sql