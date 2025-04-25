FROM ubuntu:noble
COPY gropple /

RUN apt update &&  apt install -y curl python3 ffmpeg
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/bin/yt-dlp
RUN chmod a+x /usr/bin/yt-dlp

# Run executable
CMD ["/gropple", "--config-path", "/config/gropple.json"]
