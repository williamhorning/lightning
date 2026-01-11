#!/usr/bin/env fish

set tag (git describe --tags --exact-match 2>/dev/null; or git rev-parse --short HEAD)

colima start
docker buildx create --name release --use 2>/dev/null; or docker buildx use release

set image codeberg.org/jersey/lightning
set -x SOURCE_DATE_EPOCH (git log -1 --pretty=%ct)
set date_str (date -u -r $SOURCE_DATE_EPOCH '+%Y-%m-%d %H:%M:%S')
set archs amd64 arm64

for arch in $archs
    docker buildx build . -f etc/containerfile \
        --platform linux/$arch \
        --build-arg VERSION=$tag \
        --build-arg SOURCE_DATE_STR="$date_str" \
        -t $image:$arch-$tag --load
end

read -P "push the images? (y/N): " do_push
if test "$do_push" = "y"
    read -P "codeberg token: " token; test -n "$token"; or exit 1
    echo $token | docker login codeberg.org -u jersey --password-stdin

    for arch in $archs
        docker push $image:$arch-$tag
    end

    for final_tag in $tag latest
        docker buildx imagetools create --tag $image:$final_tag $image:amd64-$tag $image:arm64-$tag
    end
end
