# Utiliser une image de base officielle de Go
FROM golang:1.22

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers nécessaires dans le conteneur
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Construire l'application
RUN go build -o server .

# Exposer le port pour le service
EXPOSE 8080

# Commande par défaut pour exécuter l'application
CMD ["./server"]
