FROM eclipse-temurin:21-jdk-alpine AS builder

WORKDIR /app

# Copy Maven wrapper and pom files
COPY mvnw pom.xml ./
COPY .mvn .mvn
COPY common/pom.xml common/
COPY chapter-01-fundamentals/pom.xml chapter-01-fundamentals/
COPY chapter-02-bloom-filters/pom.xml chapter-02-bloom-filters/
COPY chapter-03-race-conditions/pom.xml chapter-03-race-conditions/
COPY chapter-04-distributed-patterns/pom.xml chapter-04-distributed-patterns/
COPY chapter-05-consistency/pom.xml chapter-05-consistency/
COPY chapter-06-nginx-caching/pom.xml chapter-06-nginx-caching/
COPY chapter-07-read-replicas/pom.xml chapter-07-read-replicas/
COPY chapter-08-advanced-patterns/pom.xml chapter-08-advanced-patterns/

# Download dependencies
RUN ./mvnw dependency:go-offline -B

# Copy source code
COPY common/src common/src
COPY chapter-01-fundamentals/src chapter-01-fundamentals/src
COPY chapter-02-bloom-filters/src chapter-02-bloom-filters/src
COPY chapter-03-race-conditions/src chapter-03-race-conditions/src
COPY chapter-04-distributed-patterns/src chapter-04-distributed-patterns/src
COPY chapter-05-consistency/src chapter-05-consistency/src
COPY chapter-06-nginx-caching/src chapter-06-nginx-caching/src
COPY chapter-07-read-replicas/src chapter-07-read-replicas/src
COPY chapter-08-advanced-patterns/src chapter-08-advanced-patterns/src

# Build all modules
RUN ./mvnw clean package -DskipTests -B

FROM eclipse-temurin:21-jre-alpine

WORKDIR /app

# Install curl for health checks
RUN apk add --no-cache curl

# Copy built artifacts - using chapter-01 as default
COPY --from=builder /app/chapter-01-fundamentals/target/*.jar app.jar

# Default to chapter-01, can be overridden
ENV CHAPTER=chapter-01-fundamentals

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
