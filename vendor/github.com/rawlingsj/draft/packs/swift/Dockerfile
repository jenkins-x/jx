FROM swift

WORKDIR /src
ONBUILD COPY . /src
ONBUILD RUN swift build -c release

ENV PORT 8080
EXPOSE 8080

CMD ["/bin/bash", "-c", "find .build -executable -type f -not -name '*.so'| swift package describe | grep "Type: executable" | head | bash"]
