export const dockerTemplate = `
    FROM mcr.microsoft.com/dotnet/core/sdk:3.1 AS build-env

    # Copy csproj and restore as distinct layers
    #{COPY_PROJECT_DEPENDENCIES_PROJECTFILES}#

    #{COPY_PROJECTFILE}#

    WORKDIR /#{PROJECTDIR}#
    RUN dotnet restore

    # Copy everything else and build
    #{COPY_PROJECT_DEPENDENCIES}#
    #{COPY_PROJECTDIR}#

    RUN dotnet publish -c Release -o out

    # Build runtime image
    FROM #{DOCKER_RUNTIME_IMAGE}#:3.1
    WORKDIR /app
    COPY --from=build-env #{PROJECTDIR}#/out .
    ENTRYPOINT ["dotnet", "#{PROJECTNAME}#.dll"]
`

