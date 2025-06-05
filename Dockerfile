# Image officielle Go en version alpine (léger)
FROM golang:1.20-alpine

# Dossier de travail dans le conteneur
WORKDIR /app

# Copier tout le contenu local dans le conteneur
COPY . .

# Compiler le programme Go
RUN go build -o testapp .

# Commande par défaut
CMD ["./testapp"]
