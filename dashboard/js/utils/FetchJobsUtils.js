function get(url) {
  return fetch(url).then(resp => resp.json());
}

export function fetchJobs() {
  return get("/api/jobs");
}

export function fetchJob(id) {
  return get(`/api/jobs${id}`);
}

export function fetchQueueStates(times) {
  return get(`/api/stats/queues?ts=${times.join(",")}`);
}
