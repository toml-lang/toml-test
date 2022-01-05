#!/usr/bin/env python3

import glob, os.path, re

for f in glob.glob('tests/invalid/*/*.multi'):
    base = os.path.dirname(f[:-6])
    for line in open(f, 'rb').readlines():
        name = line.split(b'=')[0].strip().decode()
        if name == '' or name[0] == '#':
            continue

        line = re.sub(r'(?<=[^\\])\\x([0-9a-fA-F]{2})', lambda m: chr(int(m[1], 16)), line.decode())
        path = base + "/" + name + '.toml'
        with open(path, 'wb+') as fp:
            fp.write(line.encode())
