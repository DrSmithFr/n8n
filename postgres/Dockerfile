# This is installing the pgvector extension for postgres
FROM postgres:16

# Install requirements
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    postgresql-server-dev-all \
    && rm -rf /var/lib/apt/lists/*

# Download pgVector
WORKDIR /tmp
RUN git clone https://github.com/pgvector/pgvector.git

# Install pgVector
WORKDIR /tmp/pgvector
RUN make
RUN make install

# Loading extension
COPY ./vector_extension.sql /docker-entrypoint-initdb.d/0-vector_extension.sql