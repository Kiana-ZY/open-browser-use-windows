#!/usr/bin/env node

import { deflateRawSync } from "node:zlib";
import { mkdir, readdir, readFile, stat, writeFile } from "node:fs/promises";
import path from "node:path";

const CRC_TABLE = new Uint32Array(256);
for (let i = 0; i < CRC_TABLE.length; i += 1) {
  let value = i;
  for (let bit = 0; bit < 8; bit += 1) {
    value = value & 1 ? 0xedb88320 ^ (value >>> 1) : value >>> 1;
  }
  CRC_TABLE[i] = value >>> 0;
}

function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) {
    crc = CRC_TABLE[(crc ^ byte) & 0xff] ^ (crc >>> 8);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function dosTime(date) {
  const year = Math.max(1980, date.getFullYear());
  const dosDate = ((year - 1980) << 9) | ((date.getMonth() + 1) << 5) | date.getDate();
  const dosTimeValue = (date.getHours() << 11) | (date.getMinutes() << 5) | Math.floor(date.getSeconds() / 2);
  return { dosDate, dosTime: dosTimeValue };
}

function uint16(value) {
  const buffer = Buffer.allocUnsafe(2);
  buffer.writeUInt16LE(value);
  return buffer;
}

function uint32(value) {
  const buffer = Buffer.allocUnsafe(4);
  buffer.writeUInt32LE(value >>> 0);
  return buffer;
}

async function collectFiles(rootDir, currentDir = rootDir) {
  const entries = await readdir(currentDir, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const absolutePath = path.join(currentDir, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await collectFiles(rootDir, absolutePath)));
    } else if (entry.isFile()) {
      const relativePath = path.relative(rootDir, absolutePath).split(path.sep).join("/");
      files.push({ absolutePath, relativePath });
    }
  }

  return files;
}

async function createZip(sourceDir, outputPath) {
  const files = (await collectFiles(sourceDir)).sort((left, right) =>
    left.relativePath.localeCompare(right.relativePath)
  );
  const localParts = [];
  const centralParts = [];
  let offset = 0;

  for (const file of files) {
    const data = await readFile(file.absolutePath);
    const compressed = deflateRawSync(data, { level: 9 });
    const metadata = await stat(file.absolutePath);
    const { dosDate, dosTime: entryTime } = dosTime(metadata.mtime);
    const filename = Buffer.from(file.relativePath, "utf8");
    const checksum = crc32(data);
    const flags = 0x0800;
    const method = 8;

    if (data.length > 0xffffffff || compressed.length > 0xffffffff || offset > 0xffffffff) {
      throw new Error(`ZIP64 is not supported for ${file.relativePath}`);
    }

    const localHeader = Buffer.concat([
      uint32(0x04034b50),
      uint16(20),
      uint16(flags),
      uint16(method),
      uint16(entryTime),
      uint16(dosDate),
      uint32(checksum),
      uint32(compressed.length),
      uint32(data.length),
      uint16(filename.length),
      uint16(0),
      filename
    ]);
    localParts.push(localHeader, compressed);

    const centralHeader = Buffer.concat([
      uint32(0x02014b50),
      uint16(20),
      uint16(20),
      uint16(flags),
      uint16(method),
      uint16(entryTime),
      uint16(dosDate),
      uint32(checksum),
      uint32(compressed.length),
      uint32(data.length),
      uint16(filename.length),
      uint16(0),
      uint16(0),
      uint16(0),
      uint16(0),
      uint32(0),
      uint32(offset),
      filename
    ]);
    centralParts.push(centralHeader);
    offset += localHeader.length + compressed.length;
  }

  const centralDirectory = Buffer.concat(centralParts);
  const endOfCentralDirectory = Buffer.concat([
    uint32(0x06054b50),
    uint16(0),
    uint16(0),
    uint16(files.length),
    uint16(files.length),
    uint32(centralDirectory.length),
    uint32(offset),
    uint16(0)
  ]);

  await mkdir(path.dirname(outputPath), { recursive: true });
  await writeFile(outputPath, Buffer.concat([...localParts, centralDirectory, endOfCentralDirectory]));
}

async function listZipEntries(zipPath) {
  const data = await readFile(zipPath);
  const eocdSignature = 0x06054b50;
  let eocdOffset = -1;

  for (let index = data.length - 22; index >= 0; index -= 1) {
    if (data.readUInt32LE(index) === eocdSignature) {
      eocdOffset = index;
      break;
    }
  }

  if (eocdOffset < 0) {
    throw new Error("End of central directory record not found");
  }

  const entryCount = data.readUInt16LE(eocdOffset + 10);
  let offset = data.readUInt32LE(eocdOffset + 16);
  const entries = [];

  for (let index = 0; index < entryCount; index += 1) {
    if (data.readUInt32LE(offset) !== 0x02014b50) {
      throw new Error(`Central directory entry ${index} is invalid`);
    }
    const filenameLength = data.readUInt16LE(offset + 28);
    const extraLength = data.readUInt16LE(offset + 30);
    const commentLength = data.readUInt16LE(offset + 32);
    const filenameStart = offset + 46;
    const filenameEnd = filenameStart + filenameLength;
    entries.push(data.subarray(filenameStart, filenameEnd).toString("utf8"));
    offset = filenameEnd + extraLength + commentLength;
  }

  process.stdout.write(`${entries.join("\n")}${entries.length > 0 ? "\n" : ""}`);
}

const [command, ...args] = process.argv.slice(2);
if (command === "create" && args.length === 2) {
  await createZip(path.resolve(args[0]), path.resolve(args[1]));
} else if (command === "list" && args.length === 1) {
  await listZipEntries(path.resolve(args[0]));
} else {
  console.error("usage: zip-tool.mjs create <source-dir> <output.zip>");
  console.error("   or: zip-tool.mjs list <archive.zip>");
  process.exit(2);
}
