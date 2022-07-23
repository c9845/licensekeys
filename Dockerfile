#This builds the container we will send out to end-users. You will need to provide a
#volume for storing the config file and sqlite database file.

#To build:
#  - docker build -t licensekeys:v2.0.0 -f Dockerfile .

#To run.:
#First, run and stop the binary locally (or on the Docker host) to create a default 
#config file and empty database.
#  - This needs to be done so that the --mount flags work properly. If the config and
#    database files do not already exist, then --mount will create empty directories
#    at the given paths instead and the container/binary will not run properly.
#Next, edit the config file and set the DBPath field to /licensekeys.db
#  - This is the path the database expects to be located at in the container. You can
#    set this to something different but you will need to change the --mount flag
#    accordingly.
#Next, edit the config file and set the DBJournalMode to DELETE.
#  - If you set this to WAL, database changes will not be saved to the .db file as
#    expected. Changes are saved to the .db-shm and .db-wal files and only saved to
#    the .db file every so often. See https://sqlite.org/wal.html#automatic_checkpoint.
#  - The solution is to use DELETE mode that doesn't need these extra files since
#    changes are saved to the .db file immediately. 
#  - If you want to WAL mode, you must store the .db file in a directory (ex.: ~/lks_db)
#    and provide the directory, not the file, in the --mount call. You will also have
#    to update your config file to set the DBPath to a directory in the container as
#    well. See example changes below. Note!: You must deploy the database outside of
#    the container first so the .db, .db-shm, and .db-wal files already exist!
#    - Host system: /path/to/lbs_db/licensekeys.db
#    - DBPath: "/lks_db/licensekeys.db"
#    - Mount: --mount type=bind,source=/path/to/lks_db, target=/lks_db
#Last, run the container with the correct --mount flags set (see below).
#  - docker run \
#       -p 8007:8007 \
#       --mount type=bind,source=/path/to/licensekeys.conf, target=/licensekeys.conf \
#       --mount type=bind,source=/path/to/licensekeys.db, target=/licensekeys.db \
#       -i licensekeys:v2.0.0

#####################################################################################
#Build the binary.
#Use a 2-step build process to result in a smaller container for running the binary.
FROM golang:latest AS builder

#Copy our source code to the container. This will copy everything from the source
#repo, including not public files. However, since this container is just used for 
#building the binary, this is okay.
COPY . /licensekeys-src 
WORKDIR /licensekeys-src

#Get dependency packages.
# -v can be used for more versobe logging for diagnostics.
RUN go get -d

#Build the binary.
# -v can be used for more versobe logging for diagnostics.
# -trimpath removes absolute paths from binary and just uses relative paths in panic logging.
# -ldflags '-s -w' strips some debugging info from the binary.
# -tags 'modernc' tells the sqldb package to use the modernc sqlite library which allows for static builds (modernc doesn't use CGO).
# -o provides the path and name of the binary.
# CGO_ENABLED=0 makes sure CGO is disabled to force static builds.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags '-s -w' -tags 'modernc' -o /licensekeys

#####################################################################################
#Build the container to run the binary.
#This container just runs the binary and can be minimal.
#Notes: 
#  - Ubuntu for testing since we have access to an interactive terminal inside container with ls, file, ldd, etc. commands.
#  - Scratch for minimal production builds.
#FROM ubuntu:20.04 
FROM scratch

#Copy in the app's binary from the builder container.
WORKDIR /
COPY --from=builder /licensekeys .

#Copy in the other static files. We don't want to copy all source files, just basic
#files to power webapp. However, even these files aren't really needed since we by
#default use binary embedded HTML, CSS, and JS files.
COPY ./website /website

#Expose the default port.
EXPOSE 8007

#Run the executable.
CMD ["/licensekeys"]
# CMD ["/licensekeys", "--config", "/licensekeys.conf"]
