import sharp from 'sharp';
import { readFileSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dirname, '..');

const svgBuffer = readFileSync(resolve(root, 'app-icon.svg'));

const sizes = {
  'android/app/src/main/res/mipmap-mdpi': 48,
  'android/app/src/main/res/mipmap-hdpi': 72,
  'android/app/src/main/res/mipmap-xhdpi': 96,
  'android/app/src/main/res/mipmap-xxhdpi': 144,
  'android/app/src/main/res/mipmap-xxxhdpi': 192,
};

for (const [dir, size] of Object.entries(sizes)) {
  const outDir = resolve(root, dir);
  mkdirSync(outDir, { recursive: true });
  await sharp(svgBuffer)
    .resize(size, size)
    .png()
    .toFile(resolve(outDir, 'ic_launcher.png'));
  await sharp(svgBuffer)
    .resize(size, size)
    .png()
    .toFile(resolve(outDir, 'ic_launcher_round.png'));
  console.log(`Generated ${size}x${size} icon`);
}

// Adaptive icon foreground
const fgSize = 108;
for (const [dir, size] of Object.entries(sizes)) {
  const outDir = resolve(root, dir);
  await sharp(svgBuffer)
    .resize(fgSize, fgSize)
    .png()
    .toFile(resolve(outDir, 'ic_launcher_foreground.png'));
  console.log(`Generated foreground ${fgSize}x${fgSize} for ${dir}`);
}

console.log('All icons generated!');
