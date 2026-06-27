---
created: 2026-06-26T23:54:19 (UTC -04:00)
tags: []
source: https://www.last.fm/api/show/tag.getTopTracks
author: 
---

# API Docs | Last.fm

> ## Excerpt
> The world's largest online music service. Listen online, find out more about your favourite artists, and get music recommendations, only at Last.fm

---
## [#](https://www.last.fm/api/show/tag.getTopTracks#tag-gettoptracks) tag.getTopTracks

Get the top tracks tagged by this tag, ordered by tag count.

## [#](https://www.last.fm/api/show/tag.getTopTracks#example-urls) Example URLs

**JSON:** [/2.0/?method=tag.gettoptracks&tag=disco&api\_key=YOUR\_API\_KEY&format=json (opens new window)](http://ws.audioscrobbler.com/2.0/?method=tag.gettoptracks&tag=disco&api_key=YOUR_API_KEY&format=json)  
**XML:** [/2.0/?method=tag.gettoptracks&tag=disco&api\_key=YOUR\_API\_KEY (opens new window)](http://ws.audioscrobbler.com/2.0/?method=tag.gettoptracks&tag=disco&api_key=YOUR_API_KEY)

## [#](https://www.last.fm/api/show/tag.getTopTracks#params) Params

**tag** (Required) : The tag name  
**limit** (Optional) : The number of results to fetch per page. Defaults to 50.  
**page** (Optional) : The page number to fetch. Defaults to first page.  
**api\_key** (Required) : A Last.fm API key.

## [#](https://www.last.fm/api/show/tag.getTopTracks#auth) Auth

This service does **not** require authentication.

## [#](https://www.last.fm/api/show/tag.getTopTracks#sample-response) Sample Response

## [#](https://www.last.fm/api/show/tag.getTopTracks#errors) Errors

-   **2** : Invalid service - This service does not exist
-   **3** : Invalid Method - No method with that name in this package
-   **4** : Authentication Failed - You do not have permissions to access the service
-   **5** : Invalid format - This service doesn't exist in that format
-   **6** : Invalid parameters - Your request is missing a required parameter
-   **7** : Invalid resource specified
-   **8** : Operation failed - Something else went wrong
-   **9** : Invalid session key - Please re-authenticate
-   **10** : Invalid API key - You must be granted a valid key by last.fm
-   **11** : Service Offline - This service is temporarily offline. Try again later.
-   **13** : Invalid method signature supplied
-   **16** : There was a temporary error processing your request. Please try again
-   **26** : Suspended API key - Access for your account has been suspended, please contact Last.fm
-   **29** : Rate limit exceeded - Your IP has made too many requests in a short period
