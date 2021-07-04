FROM gradle:7.1-jdk

USER root

# add minepkg
ADD https://storage.googleapis.com/minepkg-client/latest/minepkg-linux-amd64 /usr/bin/minepkg
RUN chmod +rx /usr/bin/minepkg

RUN mkdir /etc/minepkg && echo 'useSystemJava=true\n' > /etc/minepkg/config.toml

USER gradle
CMD ["/usr/bin/minepkg"]