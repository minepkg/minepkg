FROM gradle:4.10-jdk8

USER root

# add minepkg
ADD https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64 /usr/bin/minepkg
RUN chmod +rx /usr/bin/minepkg

USER gradle

RUN echo 'useSystemJava=true\n' > /etc/minepkg/config.toml

CMD ["/usr/bin/minepkg"]