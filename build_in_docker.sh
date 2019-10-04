docker build -t 'sajid2045/jx-builder' -f Dockerfile.jx-builder .
#docker run -it --rm -v $(pwd):/src/  sajid2045/jx-builder  make darwin
docker run -it --rm -v $(pwd):/src/  sajid2045/jx-builder  make

