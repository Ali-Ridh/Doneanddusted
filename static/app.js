// State Management
const state = {
    currentUser: null,
    currentToken: null,
    currentPage: 1,
    currentSearch: '',
    currentGameFilter: '',
    currentTagFilter: '',
    selectedGameId: null,
    selectedPostId: null,
    replyingToCommentId: null
};

// DOM Ready
document.addEventListener('DOMContentLoaded', function() {
    initializeApp();
    setupEventListeners();
});

// Initialize App
function initializeApp() {
    const token = localStorage.getItem('authToken');
    if (token) {
        state.currentToken = token;
        try {
            const payload = JSON.parse(atob(token.split('.')[1]));
            state.currentUser = { id: payload.user_id, username: payload.username };
            showAuthenticatedView();
        } catch (e) {
            localStorage.removeItem('authToken');
            showUnauthenticatedView();
        }
    } else {
        showUnauthenticatedView();
    }
    
    loadPosts();
    loadTags();
    loadLocalGames(); // Load games on init
}

// Setup Event Listeners
function setupEventListeners() {
    // Navigation tabs
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.addEventListener('click', () => switchTab(tab.dataset.tab));
    });

    // Auth buttons
    document.getElementById('loginBtn').addEventListener('click', () => showAuthModal('login'));
    document.getElementById('registerBtn').addEventListener('click', () => showAuthModal('register'));
    document.getElementById('logoutBtn').addEventListener('click', logout);
    document.getElementById('closeAuthModal').addEventListener('click', hideAuthModal);
    document.getElementById('switchToRegister').addEventListener('click', (e) => {
        e.preventDefault();
        toggleAuthForm('register');
    });
    document.getElementById('switchToLogin').addEventListener('click', (e) => {
        e.preventDefault();
        toggleAuthForm('login');
    });

    // Auth form submissions
    document.getElementById('doLogin').addEventListener('click', login);
    document.getElementById('doRegister').addEventListener('click', register);

    // Search
    document.getElementById('searchPostsBtn').addEventListener('click', searchPosts);
    document.getElementById('clearSearchBtn').addEventListener('click', clearSearch);
    document.getElementById('postSearchQuery').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') searchPosts();
    });

    // RAWG Search
    document.getElementById('searchRawgBtn').addEventListener('click', searchRAWGGames);
    document.getElementById('rawgSearchQuery').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') searchRAWGGames();
    });

    // Create Game Form
    document.getElementById('createGameForm').addEventListener('submit', createLocalGame);

    // Create Post Form
    document.getElementById('createPostForm').addEventListener('submit', createPost);
    document.getElementById('postGameSearch').addEventListener('input', debounce(searchGamesForPost, 300));
    document.getElementById('postMedia').addEventListener('change', previewMedia);

    // Post Modal
    document.getElementById('closePostModal').addEventListener('click', hidePostModal);
    document.getElementById('submitComment').addEventListener('click', submitComment);

    // Close modals on outside click
    document.getElementById('authModal').addEventListener('click', (e) => {
        if (e.target.id === 'authModal') hideAuthModal();
    });
    document.getElementById('postModal').addEventListener('click', (e) => {
        if (e.target.id === 'postModal') hidePostModal();
    });
}

// Tab Navigation
function switchTab(tabName) {
    console.log('Switching to tab:', tabName);
    
    // Update nav tab buttons
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.classList.toggle('active', tab.dataset.tab === tabName);
    });
    
    // Update tab content visibility using class only (CSS handles display with !important)
    document.querySelectorAll('.tab-content').forEach(content => {
        const isActive = content.id === tabName + 'Tab';
        if (isActive) {
            content.classList.add('active');
        } else {
            content.classList.remove('active');
        }
    });

    // Load content for specific tabs
    if (tabName === 'games') {
        loadLocalGames();
    } else if (tabName === 'feed') {
        loadPosts();
    } else if (tabName === 'createPost') {
        // Ensure games are loaded for the game selector
        console.log('Create Post tab activated');
    }
}

// Auth Views
function showAuthenticatedView() {
    document.getElementById('authSection').classList.add('hidden');
    document.getElementById('userSection').classList.remove('hidden');
    document.getElementById('createTab').classList.remove('hidden');
    document.getElementById('createGameSection').classList.remove('hidden');
    document.getElementById('commentForm').classList.remove('hidden');
    document.getElementById('userDisplay').textContent = `Welcome, ${state.currentUser.username}!`;
}

function showUnauthenticatedView() {
    document.getElementById('authSection').classList.remove('hidden');
    document.getElementById('userSection').classList.add('hidden');
    document.getElementById('createTab').classList.add('hidden');
    document.getElementById('createGameSection').classList.add('hidden');
    document.getElementById('commentForm').classList.add('hidden');
}

// Auth Modal
function showAuthModal(form) {
    document.getElementById('authModal').classList.remove('hidden');
    toggleAuthForm(form);
}

function hideAuthModal() {
    document.getElementById('authModal').classList.add('hidden');
}

function toggleAuthForm(form) {
    document.getElementById('loginForm').classList.toggle('hidden', form !== 'login');
    document.getElementById('registerForm').classList.toggle('hidden', form !== 'register');
}

// Authentication
async function login() {
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    if (!username || !password) {
        alert('Please fill in all fields');
        return;
    }

    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) {
            const data = await response.json();
            state.currentToken = data.token;
            state.currentUser = data.user;
            localStorage.setItem('authToken', state.currentToken);
            hideAuthModal();
            showAuthenticatedView();
            loadPosts();
        } else {
            const error = await response.json();
            alert('Login failed: ' + error.error);
        }
    } catch (error) {
        console.error('Login error:', error);
        alert('Login failed');
    }
}

async function register() {
    const username = document.getElementById('regUsername').value;
    const email = document.getElementById('regEmail').value;
    const password = document.getElementById('regPassword').value;
    const confirmPassword = document.getElementById('regConfirmPassword').value;

    if (!username || !email || !password || !confirmPassword) {
        alert('Please fill in all fields');
        return;
    }

    if (password !== confirmPassword) {
        alert('Passwords do not match');
        return;
    }

    try {
        const response = await fetch('/api/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, email, password })
        });

        if (response.ok) {
            alert('Registration successful! Please login.');
            toggleAuthForm('login');
        } else {
            const error = await response.json();
            alert('Registration failed: ' + error.error);
        }
    } catch (error) {
        console.error('Registration error:', error);
        alert('Registration failed');
    }
}

function logout() {
    state.currentUser = null;
    state.currentToken = null;
    localStorage.removeItem('authToken');
    showUnauthenticatedView();
    loadPosts();
}

// Posts
async function loadPosts(page = 1) {
    state.currentPage = page;
    const feed = document.getElementById('postsFeed');
    feed.innerHTML = '<div class="loading">Loading posts...</div>';

    let url = `/api/posts?page=${page}&limit=10`;

    try {
        const response = await fetch(url);
        if (response.ok) {
            const data = await response.json();
            displayPosts(data.posts);
            renderPagination(data.pagination, 'postsPagination', loadPosts);
        } else {
            feed.innerHTML = '<div class="empty-state">Failed to load posts</div>';
        }
    } catch (error) {
        console.error('Posts load error:', error);
        feed.innerHTML = '<div class="empty-state">Failed to load posts</div>';
    }
}

async function searchPosts() {
    state.currentSearch = document.getElementById('postSearchQuery').value.trim();
    state.currentGameFilter = document.getElementById('gameFilter').value;
    state.currentTagFilter = document.getElementById('tagFilter').value;
    state.currentPage = 1;

    const feed = document.getElementById('postsFeed');
    feed.innerHTML = '<div class="loading">Searching...</div>';

    let url = `/api/posts/search?page=1&limit=10`;
    if (state.currentSearch) url += `&q=${encodeURIComponent(state.currentSearch)}`;
    if (state.currentGameFilter) url += `&game_id=${state.currentGameFilter}`;
    if (state.currentTagFilter) url += `&tag=${state.currentTagFilter}`;

    try {
        const response = await fetch(url);
        if (response.ok) {
            const data = await response.json();
            displayPosts(data.posts);
            renderPagination(data.pagination, 'postsPagination', (page) => {
                state.currentPage = page;
                searchPosts();
            });
        } else {
            feed.innerHTML = '<div class="empty-state">No posts found</div>';
        }
    } catch (error) {
        console.error('Search error:', error);
        feed.innerHTML = '<div class="empty-state">Search failed</div>';
    }
}

function clearSearch() {
    document.getElementById('postSearchQuery').value = '';
    document.getElementById('gameFilter').value = '';
    document.getElementById('tagFilter').value = '';
    state.currentSearch = '';
    state.currentGameFilter = '';
    state.currentTagFilter = '';
    loadPosts();
}

function displayPosts(posts) {
    const feed = document.getElementById('postsFeed');
    
    if (!posts || posts.length === 0) {
        feed.innerHTML = '<div class="empty-state">No posts found</div>';
        return;
    }

    feed.innerHTML = posts.map(post => `
        <div class="post-card" onclick="showPostDetail(${post.id})">
            <div class="post-header">
                <h3 class="post-title">${escapeHtml(post.title)}</h3>
                <span class="post-author">by ${post.user ? escapeHtml(post.user.username) : 'Anonymous'}</span>
            </div>
            ${post.game ? `<span class="post-game-tag">${escapeHtml(post.game.title)}</span>` : 
              (post.game_tag ? `<span class="post-game-tag">${escapeHtml(post.game_tag)}</span>` : '')}
            ${post.game && post.game.tags && post.game.tags.length > 0 ? `
                <div class="tags-list">
                    ${post.game.tags.slice(0, 3).map(tag => `<span class="tag">${escapeHtml(tag.name)}</span>`).join('')}
                </div>
            ` : ''}
            <div class="post-content">${escapeHtml(post.content)}</div>
            ${post.media_url ? `
                ${post.media_type === 'image' 
                    ? `<img src="${post.media_url}" alt="Post media" class="post-media">`
                    : `<video controls class="post-media"><source src="${post.media_url}" type="video/mp4"></video>`
                }
            ` : ''}
            <div class="post-footer">
                <div class="post-stats">
                    <span>💬 ${post.comment_count || 0}</span>
                </div>
                <span class="post-date">${formatDate(post.created_at)}</span>
            </div>
        </div>
    `).join('');
}

// Post Detail
async function showPostDetail(postId) {
    state.selectedPostId = postId;
    document.getElementById('postModal').classList.remove('hidden');
    document.getElementById('postDetail').innerHTML = '<div class="loading">Loading...</div>';
    document.getElementById('commentsList').innerHTML = '';

    try {
        const response = await fetch(`/api/posts/${postId}`);
        if (response.ok) {
            const post = await response.json();
            displayPostDetail(post);
            loadComments(postId);
        }
    } catch (error) {
        console.error('Post detail error:', error);
    }
}

function displayPostDetail(post) {
    document.getElementById('postDetail').innerHTML = `
        <div class="post-header">
            <h2 class="post-title">${escapeHtml(post.title)}</h2>
            <span class="post-author">by ${post.user ? escapeHtml(post.user.username) : 'Anonymous'}</span>
        </div>
        ${post.game ? `<span class="post-game-tag">${escapeHtml(post.game.title)}</span>` : 
          (post.game_tag ? `<span class="post-game-tag">${escapeHtml(post.game_tag)}</span>` : '')}
        ${post.game && post.game.tags && post.game.tags.length > 0 ? `
            <div class="tags-list">
                ${post.game.tags.map(tag => `<span class="tag">${escapeHtml(tag.name)}</span>`).join('')}
            </div>
        ` : ''}
        <div class="post-content" style="white-space: pre-wrap;">${escapeHtml(post.content)}</div>
        ${post.media_url ? `
            ${post.media_type === 'image' 
                ? `<img src="${post.media_url}" alt="Post media" class="post-media" style="max-width:100%;max-height:400px;">`
                : `<video controls class="post-media" style="max-width:100%;"><source src="${post.media_url}" type="video/mp4"></video>`
            }
        ` : ''}
        <div class="post-date">${formatDate(post.created_at)}</div>
    `;
}

function hidePostModal() {
    document.getElementById('postModal').classList.add('hidden');
    state.selectedPostId = null;
}

// Comments
async function loadComments(postId) {
    try {
        const response = await fetch(`/api/comments/post/${postId}`);
        if (response.ok) {
            const comments = await response.json();
            displayComments(comments);
        }
    } catch (error) {
        console.error('Comments load error:', error);
    }
}

function displayComments(comments, container = null) {
    const commentsList = container || document.getElementById('commentsList');
    
    if (!comments || comments.length === 0) {
        if (!container) {
            commentsList.innerHTML = '<div class="empty-state">No comments yet. Be the first to comment!</div>';
        }
        return;
    }

    commentsList.innerHTML = comments.map(comment => renderComment(comment)).join('');
}

function renderComment(comment, depth = 0) {
    const maxDepth = 4;
    const canReply = depth < maxDepth && state.currentUser;
    
    return `
        <div class="comment" data-comment-id="${comment.id}">
            <div class="comment-header">
                <span class="comment-author">${escapeHtml(comment.user?.username || 'Anonymous')}</span>
                <span class="comment-date">${formatDate(comment.created_at)}</span>
            </div>
            <div class="comment-content">${escapeHtml(comment.content)}</div>
            <div class="comment-actions">
                ${canReply ? `<button class="comment-action" onclick="showReplyForm(${comment.id})">↩️ Reply</button>` : ''}
                ${state.currentUser && state.currentUser.id === comment.user_id ? `
                    <button class="comment-action" onclick="deleteComment(${comment.id})">🗑️ Delete</button>
                ` : ''}
            </div>
            <div class="reply-form hidden" id="replyForm-${comment.id}">
                <textarea id="replyContent-${comment.id}" placeholder="Write a reply..." rows="2"></textarea>
                <div style="display: flex; gap: 10px;">
                    <button class="btn btn-primary btn-small" onclick="submitReply(${comment.id})">Reply</button>
                    <button class="btn btn-secondary btn-small" onclick="hideReplyForm(${comment.id})">Cancel</button>
                </div>
            </div>
            ${comment.replies && comment.replies.length > 0 ? `
                <div class="comment-replies">
                    ${comment.replies.map(reply => renderComment(reply, depth + 1)).join('')}
                </div>
            ` : ''}
        </div>
    `;
}

function showReplyForm(commentId) {
    document.querySelectorAll('.reply-form').forEach(form => form.classList.add('hidden'));
    document.getElementById(`replyForm-${commentId}`).classList.remove('hidden');
    state.replyingToCommentId = commentId;
}

function hideReplyForm(commentId) {
    document.getElementById(`replyForm-${commentId}`).classList.add('hidden');
    state.replyingToCommentId = null;
}

async function submitComment() {
    if (!state.currentToken || !state.selectedPostId) return;

    const content = document.getElementById('commentContent').value.trim();
    if (!content) {
        alert('Please enter a comment');
        return;
    }

    try {
        const response = await fetch('/api/comments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.currentToken}`
            },
            body: JSON.stringify({
                post_id: state.selectedPostId,
                content: content
            })
        });

        if (response.ok) {
            document.getElementById('commentContent').value = '';
            loadComments(state.selectedPostId);
        } else {
            const error = await response.json();
            alert('Failed to post comment: ' + error.error);
        }
    } catch (error) {
        console.error('Comment error:', error);
        alert('Failed to post comment');
    }
}

async function submitReply(parentId) {
    if (!state.currentToken || !state.selectedPostId) return;

    const content = document.getElementById(`replyContent-${parentId}`).value.trim();
    if (!content) {
        alert('Please enter a reply');
        return;
    }

    try {
        const response = await fetch('/api/comments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.currentToken}`
            },
            body: JSON.stringify({
                post_id: state.selectedPostId,
                parent_id: parentId,
                content: content
            })
        });

        if (response.ok) {
            hideReplyForm(parentId);
            loadComments(state.selectedPostId);
        } else {
            const error = await response.json();
            alert('Failed to post reply: ' + error.error);
        }
    } catch (error) {
        console.error('Reply error:', error);
        alert('Failed to post reply');
    }
}

async function deleteComment(commentId) {
    if (!confirm('Are you sure you want to delete this comment?')) return;

    try {
        const response = await fetch(`/api/comments/${commentId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${state.currentToken}`
            }
        });

        if (response.ok) {
            loadComments(state.selectedPostId);
        } else {
            const error = await response.json();
            alert('Failed to delete comment: ' + error.error);
        }
    } catch (error) {
        console.error('Delete comment error:', error);
        alert('Failed to delete comment');
    }
}

// Games - RAWG Search
async function searchRAWGGames() {
    const query = document.getElementById('rawgSearchQuery').value.trim();
    if (!query) {
        alert('Please enter a search term');
        return;
    }

    const container = document.getElementById('rawgResults');
    container.innerHTML = '<div class="loading">Searching RAWG database...</div>';

    try {
        const response = await fetch(`/api/games/rawg/search?q=${encodeURIComponent(query)}`);
        if (response.ok) {
            const data = await response.json();
            // RAWG returns results in a 'results' array
            const games = data.results || data;
            displayRAWGGames(games);
        } else {
            const error = await response.json();
            container.innerHTML = `<div class="empty-state">Search failed: ${error.error || 'Unknown error'}</div>`;
        }
    } catch (error) {
        console.error('RAWG search error:', error);
        container.innerHTML = '<div class="empty-state">Search failed - check console for details</div>';
    }
}

function displayRAWGGames(games) {
    const container = document.getElementById('rawgResults');
    
    if (!games || games.length === 0) {
        container.innerHTML = '<div class="empty-state">No games found</div>';
        return;
    }

    container.innerHTML = games.map(game => `
        <div class="game-card">
            ${game.background_image 
                ? `<img src="${game.background_image}" alt="${escapeHtml(game.name)}" class="game-cover">`
                : `<div class="game-cover-placeholder">🎮</div>`
            }
            <div class="game-info">
                <h4 class="game-title">${escapeHtml(game.name)}</h4>
                <div class="game-meta">
                    ${game.rating ? `<span class="game-rating">⭐ ${game.rating.toFixed(1)}</span>` : ''}
                    ${game.metacritic ? `<span>MC: ${game.metacritic}</span>` : ''}
                </div>
            </div>
            ${state.currentUser ? `
                <div class="game-actions">
                    <button class="btn btn-primary btn-small" onclick="importRAWGGame(${game.id})">Import</button>
                </div>
            ` : ''}
        </div>
    `).join('');
}

async function importRAWGGame(rawgId) {
    try {
        const response = await fetch('/api/games/rawg/import', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.currentToken}`
            },
            body: JSON.stringify({ rawg_id: rawgId })
        });

        if (response.ok) {
            alert('Game imported successfully!');
            loadLocalGames();
        } else {
            const error = await response.json();
            alert('Failed to import game: ' + error.error);
        }
    } catch (error) {
        console.error('Import error:', error);
        alert('Failed to import game');
    }
}

// Local Games
async function loadLocalGames() {
    const container = document.getElementById('localGames');
    container.innerHTML = '<div class="loading">Loading games...</div>';

    try {
        const response = await fetch('/api/games?limit=20');
        if (response.ok) {
            const data = await response.json();
            const games = data.games || data;
            displayLocalGames(games);
            updateGameFilter(games);
        } else {
            container.innerHTML = '<div class="empty-state">No games found</div>';
        }
    } catch (error) {
        console.error('Local games error:', error);
        container.innerHTML = '<div class="empty-state">Failed to load games</div>';
    }
}

function displayLocalGames(games) {
    const container = document.getElementById('localGames');
    
    if (!games || games.length === 0) {
        container.innerHTML = '<div class="empty-state">No games in the library yet. Search RAWG or add a game manually!</div>';
        return;
    }

    container.innerHTML = games.map(game => `
        <div class="game-card" onclick="filterByGame(${game.id})">
            ${game.cover_image 
                ? `<img src="${game.cover_image}" alt="${escapeHtml(game.title)}" class="game-cover">`
                : `<div class="game-cover-placeholder">🎮</div>`
            }
            <div class="game-info">
                <h4 class="game-title">${escapeHtml(game.title)}</h4>
                <div class="game-meta">
                    ${game.rating ? `<span class="game-rating">⭐ ${game.rating.toFixed(1)}</span>` : ''}
                    ${game.is_local ? '<span>📝 Local</span>' : '<span>🌐 RAWG</span>'}
                </div>
                ${game.tags && game.tags.length > 0 ? `
                    <div class="tags-list">
                        ${game.tags.slice(0, 3).map(tag => `<span class="tag">${escapeHtml(tag.name)}</span>`).join('')}
                    </div>
                ` : ''}
            </div>
        </div>
    `).join('');
}

function updateGameFilter(games) {
    const gameFilter = document.getElementById('gameFilter');
    gameFilter.innerHTML = '<option value="">All Games</option>';
    if (games && games.length > 0) {
        games.forEach(game => {
            const option = document.createElement('option');
            option.value = game.id;
            option.textContent = game.title;
            gameFilter.appendChild(option);
        });
    }
}

function filterByGame(gameId) {
    document.getElementById('gameFilter').value = gameId;
    state.currentGameFilter = gameId;
    switchTab('feed');
    searchPosts();
}

// Create Local Game
async function createLocalGame(e) {
    e.preventDefault();

    const title = document.getElementById('gameTitle').value.trim();
    const coverImage = document.getElementById('gameCover').value.trim();
    const description = document.getElementById('gameDescription').value.trim();
    const tags = document.getElementById('gameTags').value.trim();

    if (!title) {
        alert('Please enter a game title');
        return;
    }

    try {
        const response = await fetch('/api/games', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${state.currentToken}`
            },
            body: JSON.stringify({
                title,
                cover_image: coverImage,
                description,
                tags
            })
        });

        if (response.ok) {
            alert('Game added successfully!');
            document.getElementById('createGameForm').reset();
            loadLocalGames();
            loadTags();
        } else {
            const error = await response.json();
            alert('Failed to add game: ' + error.error);
        }
    } catch (error) {
        console.error('Create game error:', error);
        alert('Failed to add game');
    }
}

// Tags
async function loadTags() {
    try {
        const response = await fetch('/api/games/tags');
        if (response.ok) {
            const tags = await response.json();
            displayTags(tags);
            updateTagFilter(tags);
        }
    } catch (error) {
        console.error('Tags load error:', error);
    }
}

function displayTags(tags) {
    const container = document.getElementById('tagsCloud');
    
    if (!tags || tags.length === 0) {
        container.innerHTML = '<div class="empty-state">No tags yet</div>';
        return;
    }

    container.innerHTML = tags.map(tag => `
        <span class="tag" onclick="filterByTag('${tag.slug}')">${escapeHtml(tag.name)}</span>
    `).join('');
}

function updateTagFilter(tags) {
    const tagFilter = document.getElementById('tagFilter');
    tagFilter.innerHTML = '<option value="">All Tags</option>';
    if (tags && tags.length > 0) {
        tags.forEach(tag => {
            const option = document.createElement('option');
            option.value = tag.slug;
            option.textContent = tag.name;
            tagFilter.appendChild(option);
        });
    }
}

function filterByTag(tagSlug) {
    document.getElementById('tagFilter').value = tagSlug;
    state.currentTagFilter = tagSlug;
    switchTab('feed');
    searchPosts();
}

// Create Post
async function searchGamesForPost() {
    const query = document.getElementById('postGameSearch').value.trim();
    const suggestions = document.getElementById('gameSuggestions');
    
    if (query.length < 2) {
        suggestions.classList.add('hidden');
        return;
    }

    try {
        // Search local games first
        const localResponse = await fetch(`/api/games?limit=10`);
        let games = [];
        
        if (localResponse.ok) {
            const data = await localResponse.json();
            const localGames = data.games || data;
            games = localGames.filter(g => 
                g.title.toLowerCase().includes(query.toLowerCase())
            );
        }

        // Also search RAWG
        const rawgResponse = await fetch(`/api/games/rawg/search?q=${encodeURIComponent(query)}`);
        if (rawgResponse.ok) {
            const rawgData = await rawgResponse.json();
            const rawgGames = rawgData.results || rawgData;
            // Add RAWG games that aren't already in local
            if (rawgGames && rawgGames.length > 0) {
                rawgGames.slice(0, 5).forEach(rg => {
                    if (!games.find(g => g.title.toLowerCase() === rg.name.toLowerCase())) {
                        games.push({
                            id: null,
                            rawg_id: rg.id,
                            title: rg.name,
                            cover_image: rg.background_image,
                            is_rawg: true
                        });
                    }
                });
            }
        }

        displayGameSuggestions(games.slice(0, 8));
    } catch (error) {
        console.error('Game search error:', error);
    }
}

function displayGameSuggestions(games) {
    const suggestions = document.getElementById('gameSuggestions');
    
    if (!games || games.length === 0) {
        suggestions.classList.add('hidden');
        return;
    }

    suggestions.classList.remove('hidden');
    suggestions.innerHTML = games.map(game => `
        <div class="game-suggestion" onclick="selectGameForPost(${game.id || 'null'}, ${game.rawg_id || 'null'}, '${escapeHtml(game.title).replace(/'/g, "\\'")}', '${game.cover_image || ''}')">
            ${game.cover_image 
                ? `<img src="${game.cover_image}" alt="${escapeHtml(game.title)}">`
                : '<div style="width:60px;height:40px;background:#333;display:flex;align-items:center;justify-content:center;">🎮</div>'
            }
            <div class="game-suggestion-info">
                <h4>${escapeHtml(game.title)}</h4>
                <span>${game.is_rawg ? '🌐 RAWG' : '📚 Local'}</span>
            </div>
        </div>
    `).join('');
}

async function selectGameForPost(gameId, rawgId, title, coverImage) {
    document.getElementById('gameSuggestions').classList.add('hidden');
    document.getElementById('postGameSearch').value = '';
    
    // If it's a RAWG game, import it first
    if (!gameId && rawgId) {
        try {
            const response = await fetch('/api/games/rawg/import', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${state.currentToken}`
                },
                body: JSON.stringify({ rawg_id: rawgId })
            });

            if (response.ok) {
                const game = await response.json();
                gameId = game.id;
            } else {
                alert('Failed to import game');
                return;
            }
        } catch (error) {
            console.error('Import error:', error);
            return;
        }
    }

    state.selectedGameId = gameId;
    document.getElementById('selectedGameId').value = gameId;
    
    const display = document.getElementById('selectedGameDisplay');
    display.classList.remove('hidden');
    display.innerHTML = `
        ${coverImage 
            ? `<img src="${coverImage}" alt="${escapeHtml(title)}">`
            : '<div style="width:80px;height:50px;background:#333;display:flex;align-items:center;justify-content:center;">🎮</div>'
        }
        <div class="selected-game-info">
            <h4>${escapeHtml(title)}</h4>
        </div>
        <button type="button" class="btn-remove" onclick="clearSelectedGame()">×</button>
    `;
}

function clearSelectedGame() {
    state.selectedGameId = null;
    document.getElementById('selectedGameId').value = '';
    document.getElementById('selectedGameDisplay').classList.add('hidden');
}

function previewMedia() {
    const file = document.getElementById('postMedia').files[0];
    const preview = document.getElementById('mediaPreview');
    
    if (!file) {
        preview.classList.add('hidden');
        return;
    }

    preview.classList.remove('hidden');
    const url = URL.createObjectURL(file);
    
    if (file.type.startsWith('image/')) {
        preview.innerHTML = `<img src="${url}" alt="Preview">`;
    } else if (file.type.startsWith('video/')) {
        preview.innerHTML = `<video controls><source src="${url}" type="${file.type}"></video>`;
    }
}

async function createPost(e) {
    e.preventDefault();

    if (!state.currentToken) {
        alert('Please login to create posts');
        return;
    }

    const title = document.getElementById('postTitle').value.trim();
    const content = document.getElementById('postContent').value.trim();
    const gameId = document.getElementById('selectedGameId').value;

    if (!title || !content) {
        alert('Please fill in title and content');
        return;
    }

    if (!gameId) {
        alert('Please select a game');
        return;
    }

    const formData = new FormData();
    formData.append('title', title);
    formData.append('content', content);
    formData.append('game_id', gameId);

    const fileInput = document.getElementById('postMedia');
    if (fileInput.files[0]) {
        formData.append('file', fileInput.files[0]);
    }

    try {
        const response = await fetch('/api/posts', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${state.currentToken}`
            },
            body: formData
        });

        if (response.ok) {
            alert('Post created successfully!');
            document.getElementById('createPostForm').reset();
            clearSelectedGame();
            document.getElementById('mediaPreview').classList.add('hidden');
            switchTab('feed');
            loadPosts();
        } else {
            const error = await response.json();
            alert('Failed to create post: ' + error.error);
        }
    } catch (error) {
        console.error('Create post error:', error);
        alert('Failed to create post');
    }
}

// Pagination
function renderPagination(pagination, containerId, callback) {
    const container = document.getElementById(containerId);
    
    if (!pagination || pagination.pages <= 1) {
        container.classList.add('hidden');
        return;
    }

    container.classList.remove('hidden');
    const { page, pages } = pagination;

    let html = '';

    if (page > 1) {
        html += `<button onclick="loadPosts(${page - 1})">Previous</button>`;
    }

    const startPage = Math.max(1, page - 2);
    const endPage = Math.min(pages, page + 2);

    for (let i = startPage; i <= endPage; i++) {
        html += `<button class="${i === page ? 'active' : ''}" onclick="loadPosts(${i})">${i}</button>`;
    }

    if (page < pages) {
        html += `<button onclick="loadPosts(${page + 1})">Next</button>`;
    }

    container.innerHTML = html;
}

// Utility Functions
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateString) {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}