require('dotenv').config();
const fs = require('fs').promises;
const path = require('path');
const { scrapeNowPlaying, scrapeMovieDetails, generateReport } = require('./scraper');
const { fetchImdbDetails } = require('./omdb');

async function main() {
    try {
        console.log('🚀 Starting ForumCinemas Movie Reporter');

        if (!process.env.OMDB_API_KEY) {
            throw new Error('OMDB_API_KEY is not configured');
        }
        
        // Step 1: Get list of movies currently playing
        console.log('📽️  Fetching now playing movies...');
        const movieLinks = await scrapeNowPlaying();
        console.log(`Found ${movieLinks.length} movies playing`);
        
        // Save intermediate data
        await fs.writeFile('data/now_playing.json', JSON.stringify(movieLinks, null, 2));
        
        // Step 2: Scrape details for each movie
        console.log('🔍 Scraping movie details...');
        const movies = [];
        
        for (let i = 0; i < movieLinks.length; i++) {
            const link = movieLinks[i];
            console.log(`Processing ${i + 1}/${movieLinks.length}: ${link.title}`);
            
            try {
                const movieDetails = await scrapeMovieDetails(link.url, link.url);
                
                if (movieDetails.imdbId) {
                    console.log(`  🎭 Fetching OMDb details for ${movieDetails.imdbId}...`);
                    const imdbData = await fetchImdbDetails(movieDetails.imdbId);
                    movieDetails.imdbRating = imdbData.rating ?? null;
                    if (imdbData.year) {
                        movieDetails.releaseYear = imdbData.year;
                        console.log(`    📅 Updated release year to ${imdbData.year} from OMDb`);
                    }
                    if (imdbData.genres && imdbData.genres.length) {
                        movieDetails.genres = imdbData.genres.join(', ');
                    }
                    if (i < movieLinks.length - 1) {
                        await new Promise(resolve => setTimeout(resolve, 200));
                    }
                } else {
                    console.log(`  ⚠️  No IMDb link found, skipping`);
                    continue;
                }
                
                movies.push(movieDetails);
            } catch (error) {
                console.error(`  ❌ Error processing ${link.title}:`, error.message);
            }
        }
        
        // Save intermediate data
        await fs.writeFile('data/movies_enriched.json', JSON.stringify(movies, null, 2));
        
        // Step 3: Generate the report
        console.log('📄 Generating HTML report...');
        await generateReport(movies);
        
        console.log(`✅ Report generated successfully! Found ${movies.length} movies with IMDb data`);
        console.log('📁 Check index.html for the final report');
        
    } catch (error) {
        console.error('❌ Error in main process:', error);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}

module.exports = { main };
