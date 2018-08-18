FROM scratch
EXPOSE 8053/tcp
EXPOSE 8053/udp
COPY target/gyip /gyip
ENTRYPOINT ["/gyip","--host 0.0.0.0"]