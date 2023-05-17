#!/usr/bin/env python3

import argparse, pathlib, shutil, re, subprocess, os, tempfile, glob, os.path

ROOT = pathlib.Path(__file__).parent
VALID_ROOT = ROOT / f"tests/valid/spec"
INVALID_ROOT = ROOT / f"tests/invalid/spec"

def gen_multi():
    for f in glob.glob(str(ROOT / 'tests/invalid/*/*.multi')):
        base = os.path.dirname(f[:-6])
        for line in open(f, 'rb').readlines():
            name = line.split(b'=')[0].strip().decode()
            if name == '' or name[0] == '#':
                continue

            line = re.sub(r'(?<=[^\\])\\x([0-9a-fA-F]{2})', lambda m: chr(int(m[1], 16)), line.decode())
            path = base + "/" + name + '.toml'
            with open(path, 'wb+') as fp:
                fp.write(line.encode())

def gen_list():
    with open('tests/files-toml-1.0.0', 'w+') as fp:
        subprocess.run(['go', 'run', './cmd/toml-test', '-list-files', '-toml=1.0.0'], stdout=fp)
    with open('tests/files-toml-1.1.0', 'w+') as fp:
        subprocess.run(['go', 'run', './cmd/toml-test', '-list-files', '-toml=1.1.0'], stdout=fp)

def gen_spec(tmp):
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--input",
        metavar="MD",
        type=pathlib.Path, default=ROOT / "assets/specs/en/v1.0.0.md",
        help="Spec to parse for test cases",
    )
    args = parser.parse_args()

    decoder = os.path.join(tmp, 'toml-test-decoder')
    subprocess.run(['go', 'build', '-o', decoder, 'github.com/BurntSushi/toml/cmd/toml-test-decoder'])

    try:
        shutil.rmtree(VALID_ROOT)
    except FileNotFoundError:
        pass
    try:
        shutil.rmtree(INVALID_ROOT)
    except FileNotFoundError:
        pass

    markdown = args.input.read_text()
    lines = markdown.splitlines()

    header = "common"
    case_index = 0

    line_index = 0
    while line_index < len(lines):
        try:
            line_index, header = parse_header(line_index, lines)
        except ParseError:
            pass
        else:
            case_index = 0
            continue

        try:
            line_index, info, block = parse_block(line_index, lines)
        except ParseError:
            pass
        else:
            if info in ["toml", ""] and block.startswith("# INVALID"):
                write_invalid_case(header, case_index, block)
                case_index += 1
            elif info == "toml":
                if has_active_invalid(block):
                    write_invalid_case(header, case_index, block)
                else:
                    write_valid_case(decoder, header, case_index, block)
                case_index += 1
            continue

        line_index += 1

class ParseError(RuntimeError):
    pass

def parse_header(line_index, lines):
    try:
        header = lines[line_index]
        if not header:
            raise ParseError()

        line_index += 1
        dashes = lines[line_index]
        if not re.fullmatch("-+", dashes):
            raise ParseError()

        line_index += 1
        blank = lines[line_index]
        if blank:
            raise ParseError()

        line_index += 1
    except IndexError:
        raise ParseError()

    header = header.lower().replace(" ", "-").replace("/", "-")
    return line_index, header

def parse_block(line_index, lines):
    info = ""
    try:
        fence = lines[line_index]
        if not fence.startswith('```'):
            raise ParseError()
        info = fence.removeprefix('```')

        block = []
        line = ""
        while line != '```':
            block.append(line)
            line_index += 1
            line = lines[line_index]

        line_index += 1
    except IndexError:
        raise ParseError()

    return line_index, info, "\n".join(block)

def write_invalid_case(header, index, block):
    path = INVALID_ROOT / f"{header}-{index}.toml"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(block.strip() + '\n')

def write_valid_case(decoder, header, index, block):
    path = VALID_ROOT / f"{header}-{index}.toml"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(block.strip() + '\n')

    subprocess.run([decoder],
        stdin=open(path),
        stdout=open(VALID_ROOT / f"{header}-{index}.json", mode='w'))

    invalid_index = 0
    lines = block.splitlines()
    for i, line in enumerate(lines):
        if "# INVALID" in line:
            new_lines = lines[:]
            assert line.startswith("# "), f"{line}"
            new_lines[i] = line.removeprefix("# ")
            write_invalid_case(header, f"{index}-{invalid_index}", "\n".join(new_lines))
            invalid_index += 1

def has_active_invalid(block):
    lines = block.splitlines()
    for line in lines:
        if "# INVALID" in line and not line.startswith("# "):
            return True
    return False

if __name__ == "__main__":
    with tempfile.TemporaryDirectory() as tmp:
        gen_spec(tmp)
    gen_multi()
    gen_list()
