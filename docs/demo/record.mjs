import { execFileSync, spawn } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "..", "..");
const port = Number(process.env.DEMO_PORT ?? 4117);
const url = `http://127.0.0.1:${port}/`;
const viewport = { width: 480, height: 860 };
const sample = join(here, "sample-export.csv");
const features = ["shiny", "costume", "background", "dynamax", "gigantamax"];

const cursorScript = `
  window.addEventListener('DOMContentLoaded', () => {
    const dot = document.createElement('div');
    dot.style.cssText = 'position:fixed;left:-40px;top:-40px;width:26px;height:26px;margin:-13px 0 0 -13px;border-radius:50%;border:2px solid rgba(255,255,255,0.9);box-sizing:border-box;background:rgba(13,148,136,0.25);box-shadow:0 0 0 2px rgba(0,0,0,0.4);pointer-events:none;z-index:99999;transition:transform 120ms';
    document.body.appendChild(dot);
    window.addEventListener('mousemove', (e) => { dot.style.left = e.clientX + 'px'; dot.style.top = e.clientY + 'px'; });
    window.addEventListener('mousedown', () => { dot.style.transform = 'scale(0.7)'; });
    window.addEventListener('mouseup', () => { dot.style.transform = 'scale(1)'; });
  });
`;

const pause = (page, ms) => page.waitForTimeout(ms);

async function pointAt(page, locator) {
  await locator.scrollIntoViewIfNeeded();
  const box = await locator.boundingBox();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2, {
    steps: 20,
  });
  await pause(page, 300);
}

async function clickOn(page, locator) {
  await pointAt(page, locator);
  await locator.click();
}

async function goTo(page, place) {
  const link = page
    .getByRole("navigation", { name: "Main" })
    .getByRole("link", { name: place });
  await clickOn(page, link);
  await page.getByRole("heading", { level: 2 }).first().waitFor();
  await pause(page, 600);
}

async function scrollBy(page, dy) {
  for (let i = 0; i < 10; i++) {
    await page.mouse.wheel(0, dy / 10);
    await pause(page, 40);
  }
}

const pin = "2468";

async function createProfile(request, name) {
  const created = await request.post(`${url}api/v1/profiles`, {
    data: { displayName: name, pin },
  });
  return (await created.json()).id;
}

async function signIn(request, name) {
  const profileId = await createProfile(request, name);
  await request.post(`${url}api/v1/session`, { data: { profileId, pin } });
}

async function signInOnScreen(page, name, delay) {
  await page.getByRole("button", { name, exact: true }).click({ delay });
  await page.getByLabel(`PIN for ${name}`).pressSequentially(pin, { delay });
  await page.getByRole("button", { name: "Sign in" }).click({ delay });
  await page.getByRole("heading", { name: `Hi, ${name}` }).waitFor();
}

async function importSample(request) {
  const preview = await request.post(`${url}api/v1/imports`, {
    headers: { "Content-Type": "text/csv" },
    data: readFileSync(sample),
  });
  const { id } = await preview.json();
  await request.post(`${url}api/v1/imports/${id}/commit`);
  await request.post(`${url}api/v1/collection/confirmations`, {
    data: { features },
  });
}

const scenes = {
  import: {
    async seed(request) {
      await createProfile(request, "Ash");
    },
    async play(page) {
      await pause(page, 800);
      await pointAt(
        page,
        page.getByRole("button", { name: "Ash", exact: true }),
      );
      await signInOnScreen(page, "Ash", 150);
      await pause(page, 1000);
      const choose = page.getByLabel("Choose your Poke Genie export");
      await pointAt(page, page.getByText("Choose your Poke Genie export"));
      await choose.setInputFiles(sample);
      const preview = page.getByRole("region", { name: "Import preview" });
      await preview.waitFor();
      await pause(page, 1800);
      await clickOn(
        page,
        preview.getByRole("button", { name: /^Add \d+ Pokémon$/ }),
      );
      await pause(page, 1200);
      await goTo(page, "Collection");
      await page.getByRole("list", { name: "Collection" }).waitFor();
      await pause(page, 1000);
      await scrollBy(page, 900);
      await pause(page, 1600);
    },
  },
  keep: {
    async seed(request) {
      await signIn(request, "Misty");
      await importSample(request);
    },
    async play(page) {
      await pause(page, 600);
      await goTo(page, "Rules");
      const setup = page.getByRole("region", {
        name: "Your storage and goals",
      });
      await setup.waitFor();
      await setup.scrollIntoViewIfNeeded();
      await pause(page, 1000);
      await clickOn(page, setup.getByLabel("Storage slots"));
      await page.keyboard.type("4000", { delay: 160 });
      await pause(page, 600);
      await clickOn(page, setup.getByRole("checkbox", { name: "Collecting" }));
      await clickOn(page, setup.getByRole("checkbox", { name: "Trading" }));
      await pause(page, 1800);
      await clickOn(page, setup.getByRole("button", { name: "Save" }));
      const keep = page.getByRole("region", { name: "Your keep rules" });
      await keep.waitFor();
      await keep.scrollIntoViewIfNeeded();
      await pause(page, 1500);
      await clickOn(page, keep.getByRole("button", { name: "Got it" }));
      const pidgey = keep.getByRole("button", { name: /^Pidgey/ });
      await pointAt(page, pidgey);
      await pause(page, 600);
      await pidgey.click();
      await pause(page, 600);
      await keep
        .getByRole("region", { name: /^Why Pidgey keeps/ })
        .scrollIntoViewIfNeeded();
      await pause(page, 2500);
    },
  },
  protect: {
    async seed(request) {
      await signIn(request, "Brock");
      await importSample(request);
      await request.put(`${url}api/v1/setup`, {
        data: { capacity: 4000, goals: ["collecting"] },
      });
    },
    async play(page) {
      await pause(page, 600);
      await goTo(page, "Rules");
      const keep = page.getByRole("region", { name: "Your keep rules" });
      await keep.waitFor();
      const pidgey = keep.getByRole("button", { name: /^Pidgey/ });
      await pidgey.scrollIntoViewIfNeeded();
      await pause(page, 800);
      await clickOn(page, pidgey);
      const copies = keep.getByRole("list", { name: "Pidgey copies" });
      await copies.scrollIntoViewIfNeeded();
      await scrollBy(page, 200);
      await pause(page, 1500);
      const weakest = copies.getByRole("button", { name: /^Protect Pidgey/ });
      await clickOn(page, weakest.last());
      await pause(page, 2200);
      await keep
        .getByRole("region", { name: /^Why Pidgey keeps/ })
        .scrollIntoViewIfNeeded();
      await pause(page, 2500);
    },
  },
  room: {
    async seed(request) {
      await signIn(request, "Dawn");
      await importSample(request);
      await request.put(`${url}api/v1/setup`, {
        data: { capacity: 50, goals: ["collecting"], inGameCount: 45 },
      });
    },
    async play(page) {
      await pause(page, 800);
      await goTo(page, "Make room");
      const room = page.getByRole("region", { name: "Make room" });
      await room.getByRole("heading", { name: /^Transfer/ }).waitFor();
      await pause(page, 1800);
      await clickOn(page, room.getByRole("radio", { name: /Free more slots/ }));
      await clickOn(page, room.getByLabel("Slots to free"));
      await page.keyboard.type("8", { delay: 200 });
      await pause(page, 1500);
      const find = room.getByRole("region", { name: "Find them in the game" });
      await find.scrollIntoViewIfNeeded();
      await pause(page, 2200);
      await scrollBy(page, 700);
      await pause(page, 2000);
    },
  },
};

function toGif(video, offsetSeconds, out) {
  execFileSync("ffmpeg", [
    "-y",
    "-loglevel",
    "error",
    "-ss",
    offsetSeconds.toFixed(2),
    "-i",
    video,
    "-vf",
    "fps=6,scale=480:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=48:stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle",
    out,
  ]);
}

async function launch() {
  const channel = process.env.DEMO_CHANNEL;
  if (channel) return chromium.launch({ channel });
  try {
    return await chromium.launch();
  } catch {
    return chromium.launch({ channel: "chrome" });
  }
}

async function waitFor(target, tries = 50) {
  for (let i = 0; i < tries; i++) {
    try {
      if ((await fetch(target)).ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error(`server not ready at ${target}`);
}

const data = mkdtempSync(join(tmpdir(), "catchkeep-demo-"));
const videoDir = mkdtempSync(join(tmpdir(), "catchkeep-video-"));
const server = spawn(
  join(root, "backend", "bin", "catchkeep"),
  [
    "serve",
    "-addr",
    `127.0.0.1:${port}`,
    "-db",
    join(data, "demo.db"),
    "-web-dir",
    join(root, "frontend", "dist"),
  ],
  { stdio: "ignore" },
);
process.on("exit", () => server.kill());
const only = process.argv.slice(2);
let browser;

try {
  await waitFor(`${url}api/v1/health`);
  browser = await launch();
  for (const [name, scene] of Object.entries(scenes)) {
    if (only.length > 0 && !only.includes(name)) continue;
    const context = await browser.newContext({
      viewport,
      colorScheme: "light",
      recordVideo: { dir: videoDir, size: viewport },
    });
    await context.addInitScript(cursorScript);
    await scene.seed(context.request);
    const started = Date.now();
    const page = await context.newPage();
    await page.goto(url);
    await page.waitForLoadState("networkidle");
    const offset = (Date.now() - started) / 1000;
    await scene.play(page);
    await context.close();
    toGif(await page.video().path(), offset, join(here, `${name}.gif`));
    console.log(`recorded docs/demo/${name}.gif`);
  }
} finally {
  await browser?.close();
  server.kill();
  rmSync(videoDir, { recursive: true, force: true });
  rmSync(data, { recursive: true, force: true });
}
