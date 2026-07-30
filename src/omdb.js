function parseOmdbResponse(data) {
  if (!data || data.Response === 'False') {
    throw new Error(data?.Error || 'OMDb returned an invalid response');
  }

  const parsedRating = data.imdbRating && data.imdbRating !== 'N/A'
    ? Number.parseFloat(data.imdbRating)
    : null;
  const yearMatch = String(data.Year || '').match(/\d{4}/);
  const genres = data.Genre && data.Genre !== 'N/A'
    ? data.Genre.split(',').map(genre => genre.trim()).filter(Boolean)
    : [];

  return {
    rating: Number.isFinite(parsedRating) ? parsedRating : null,
    year: yearMatch ? Number.parseInt(yearMatch[0], 10) : null,
    genres
  };
}

async function fetchImdbDetails(imdbId, opts = {}) {
  const axios = require('axios');
  const apiKey = opts.apiKey || process.env.OMDB_API_KEY;
  if (!apiKey) {
    throw new Error('OMDB_API_KEY is not configured');
  }

  if (!/^tt\d+$/.test(imdbId)) {
    throw new Error(`Invalid IMDb ID: ${imdbId}`);
  }

  const response = await axios.get('https://www.omdbapi.com/', {
    params: {
      apikey: apiKey,
      i: imdbId,
      r: 'json'
    },
    timeout: opts.timeoutMs ?? 10000
  });
  const parsed = parseOmdbResponse(response.data);

  return {
    ...parsed,
    imdbUrl: `https://www.imdb.com/title/${imdbId}/`
  };
}

module.exports = { fetchImdbDetails, parseOmdbResponse };
