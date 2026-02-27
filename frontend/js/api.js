async function apiRequest(url, method = 'GET', body = null) {
  const options = {
    method,
    headers: {
      'Content-Type': 'application/json',
      // Add 'Authorization': 'Bearer token' later for auth
    }
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  try {
    const response = await fetch(`http://localhost:8080${url}`, options);  // Your Go server
    if (!response.ok) throw new Error('API Error');
    return await response.json();
  } catch (error) {
    console.error(error);
  }
}

// Example: Get nearby requests
async function getRequests() {
  const data = await apiRequest('/requests');
  // Render to feed
}

// Example: Post new request
async function postRequest(formData) {
  await apiRequest('/requests', 'POST', formData);
}