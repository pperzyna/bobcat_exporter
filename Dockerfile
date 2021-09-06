FROM scratch
EXPOSE 9857
USER 1000

COPY bobcat_exporter /bin/bobcat_exporter

ENTRYPOINT ["bobcat_exporter"]
