#!/usr/bin/env python3

import glob

for f in glob.glob('tests/invalid/*.multi'):
    base = f[:-6] + '-'
    for l in open(f, 'rb').readlines():
        name = l.split(b'=')[0].strip().decode()
        if name == '' or name[0] == '#':
            continue
        with open(base + name + '.toml', 'wb+') as fp:
            fp.write(l)
