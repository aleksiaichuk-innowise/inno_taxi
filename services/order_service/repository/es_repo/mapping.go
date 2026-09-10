package es_repo

// indexMapping is the Elasticsearch index-creation body for the orders
// index - keyword fields are exact-match filtered on, date/long/integer
// fields support range filters, the rest are stored for document fidelity
// only (see the design doc's "Data flow: indexing" mapping table).
const indexMapping = `{
  "mappings": {
    "properties": {
      "id": {"type": "keyword"},
      "user_id": {"type": "keyword"},
      "driver_id": {"type": "keyword"},
      "taxi_type": {"type": "keyword"},
      "status": {"type": "keyword"},
      "price_minor_units": {"type": "long"},
      "rating": {"type": "integer"},
      "comment": {"type": "text"},
      "start_lat": {"type": "double"},
      "start_lng": {"type": "double"},
      "destination_lat": {"type": "double"},
      "destination_lng": {"type": "double"},
      "created_at": {"type": "date"},
      "updated_at": {"type": "date"}
    }
  }
}`
