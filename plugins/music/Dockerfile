# Use an official Node.js runtime as a parent image
FROM node:18-alpine

# Set the working directory in the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json files into the container
COPY plugins/music/package*.json ./

# Install app dependencies
RUN npm ci

# Copy the rest of your application code into the container
COPY plugins/music/. ./

# Set the NODE_ENV environment variable to "production"
ENV NODE_ENV=production

# Expose port 80
EXPOSE 80

# Start the app
CMD [ "npm", "start" ]