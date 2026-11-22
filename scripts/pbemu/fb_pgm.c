/* fb_pgm: write the reader's current framebuffer as a PGM (P5) file. Usage: fb_pgm out.pgm */
#define _GNU_SOURCE
#include "frame_source.h"
#include "inkinternal.h"
#include "pb_informer.h"

#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <sys/shm.h>

int
main(int argc, char **argv)
{
    informer_ctx      ctx;
    pb_informer_state state;
    framebuffer_info  info;
    struct shmid_ds   ds;
    const uint8_t    *base;
    FILE             *f;
    int               shmid, y;

    if (argc < 2) {
        fprintf(stderr, "usage: fb_pgm out.pgm\n");
        return 2;
    }
    informer_ctx_init(&ctx);
    if (read_informer_state(&ctx, &state) != 0 || !decode_informer_framebuffer_info(&state, &info)) {
        fprintf(stderr, "fb_pgm: no framebuffer yet\n");
        return 1;
    }
    informer_ctx_close(&ctx);
    shmid = shmget((key_t)state.fbinfo.shmkey, 0, 0);
    if (shmid < 0 || shmctl(shmid, IPC_STAT, &ds) != 0) {
        fprintf(stderr, "fb_pgm: no shm: %s\n", strerror(errno));
        return 1;
    }
    base = shmat(shmid, NULL, SHM_RDONLY);
    if (base == (void *)-1)
        return 1;
    f = fopen(argv[1], "wb");
    if (f == NULL)
        return 1;
    { int bpp = info.depth >= 24 ? 3 : 1;
    fprintf(f, "P%d\n%d %d\n255\n", bpp == 3 ? 6 : 5, info.width, info.height);
    for (y = 0; y < info.height; y++)
        fwrite(base + info.pixel_data_offset + (size_t)y * info.stride, 1, (size_t)info.width * bpp, f); }
    fclose(f);
    printf("task=%d %dx%d depth=%d colormask=%d stride=%d\n", state.active_task_id, info.width, info.height,
           info.depth, info.colormask, info.stride);
    return 0;
}
