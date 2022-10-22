FROM ubuntu:latest

# install ca certificates
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

# add minepkg
COPY minepkg /usr/bin/minepkg
RUN chmod +rx /usr/bin/minepkg

# set it as cmd
CMD ["/usr/bin/minepkg"]
