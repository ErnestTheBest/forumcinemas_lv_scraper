const { chromium } = require('playwright');

async function withBrowser(fn) {
  // Try to use system Chrome if available, fallback to bundled browser
  const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
  const launchOptions = { 
    headless: true,
    channel: 'chrome'
  };
  
  if (executablePath) {
    launchOptions.executablePath = executablePath;
    console.log(`🔧 Using system Chrome at: ${executablePath}`);
  } else {
    console.log('🔧 Using bundled Playwright browser');
  }
  
  const browser = await chromium.launch(launchOptions);
  const context = await browser.newContext({
    userAgent:
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36'
  });
  const page = await context.newPage();
  try {
    return await fn(page);
  } finally {
    await context.close();
    await browser.close();
  }
}

function parseJsonLd(jsonText) {
  try {
    const data = JSON.parse(jsonText);
    const obj = Array.isArray(data)
      ? data.find(
          x => x['@type'] === 'Movie' || x['@type'] === 'TVSeries' || x['@type'] === 'VideoObject'
        ) || data[0]
      : data;
    if (!obj) return null;

    const rating = obj.aggregateRating?.ratingValue ? parseFloat(obj.aggregateRating.ratingValue) : null;

    let year = null;
    const datePublished = obj.datePublished || obj.dateCreated || obj.releaseDate;
    if (datePublished) {
      const m = String(datePublished).match(/(\d{4})/);
      if (m) year = parseInt(m[1], 10);
    }

    let genres = [];
    if (Array.isArray(obj.genre)) {
      genres = obj.genre.map(g => String(g).trim()).filter(Boolean);
    } else if (typeof obj.genre === 'string') {
      genres = obj.genre
        .split(',')
        .map(g => g.trim())
        .filter(Boolean);
    }

    return { rating, year, genres };
  } catch {
    return null;
  }
}

async function fetchImdbDetails(imdbId, opts = {}) {
  const url = `https://www.imdb.com/title/${imdbId}/`;
  const timeoutMs = opts.timeoutMs ?? 20000;

  return await withBrowser(async page => {
    await page.goto(url, { timeout: timeoutMs, waitUntil: 'domcontentloaded' });

    const jsonLd = await page
      .locator('script[type="application/ld+json"]')
      .first()
      .textContent()
      .catch(() => null);
    let parsed = jsonLd ? parseJsonLd(jsonLd) : null;

    if (!parsed || (!parsed.rating && !(parsed.genres && parsed.genres.length))) {
      const ratingText = await page
        .locator('[data-testid="hero-rating-bar__aggregate-rating__score"]')
        .first()
        .textContent()
        .catch(() => null);
      const rating = ratingText ? parseFloat(ratingText.trim()) : parsed?.rating ?? null;

      const genreLoc = page.locator('[data-testid="genres"] a, [data-testid="genres"] span.ipc-chip__text');
      const genres = await genreLoc
        .allTextContents()
        .then(arr => arr.map(t => t.trim()).filter(Boolean))
        .catch(() => []);

      const yearText = await page
        .locator(
          'a[href*="releaseinfo"], a[data-testid="title-details-releasedate"], [data-testid="hero-title-block__metadata"] li a'
        )
        .first()
        .textContent()
        .catch(() => null);
      const year = yearText
        ? yearText.match(/(\d{4})/)
          ? parseInt(yearText.match(/(\d{4})/)[1], 10)
          : parsed?.year ?? null
        : parsed?.year ?? null;

      parsed = {
        rating: rating ?? parsed?.rating ?? null,
        year: year ?? parsed?.year ?? null,
        genres: genres.length ? genres : parsed?.genres ?? []
      };
    }

    return {
      rating: parsed?.rating ?? null,
      year: parsed?.year ?? null,
      genres: parsed?.genres ?? [],
      imdbUrl: url
    };
  });
}

module.exports = { fetchImdbDetails };


