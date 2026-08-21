FROM gradle:8.9-jdk21 AS build
WORKDIR /src
COPY sandbox/ ./sandbox/
COPY proto/ ./proto/
WORKDIR /src/sandbox
RUN gradle build -x test --no-daemon

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /src/sandbox/build/libs/*.jar /app/tsunagu-sandbox.jar
EXPOSE 50051
ENTRYPOINT ["java", "-jar", "/app/tsunagu-sandbox.jar"]
a