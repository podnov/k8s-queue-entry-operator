FROM ubuntu:latest
RUN useradd -u 10001 scratchuser

FROM scratch
COPY --from=0 /etc/passwd /etc/passwd
ADD main /
USER scratchuser
CMD ["/main"]

