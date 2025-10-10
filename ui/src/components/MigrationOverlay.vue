<template>
  <div class="modal is-active" v-if="isMigrating">
    <div class="modal-background"></div>
    <div class="modal-card">
      <section class="modal-card-body has-text-centered">
        <div class="content">
          <h3 class="title is-4">Database Maintenance in Progress</h3>
          <p class="subtitle is-6">Please wait while the database is being updated...</p>

          <div v-if="migrationState.total > 0" class="migration-progress">
            <p v-if="migrationPhase" class="migration-phase">{{ migrationPhase }}</p>
            <div class="progress-info">
              <span class="progress-count">{{ migrationState.progress }}/{{ migrationState.total }}</span>
              <span v-if="showETA" class="eta-text">
                ETA: {{ estimatedTimeRemaining || '∞' }}
              </span>
            </div>
            <progress
              class="progress is-primary"
              :value="migrationState.progress"
              :max="migrationState.total"
            >
              {{ progressPercent }}%
            </progress>
          </div>

          <div v-else>
            <progress class="progress is-primary" max="100"></progress>
          </div>

          <p class="help-notice mt-4">
            Web interface will become available when operation is complete.
          </p>
        </div>
      </section>
    </div>
  </div>
</template>

<script>
import ky from 'ky'

export default {
  name: 'MigrationOverlay',
  data() {
    return {
      isMigrating: false,
      migrationState: {
        is_running: false,
        current: '',
        total: 0,
        progress: 0,
        message: ''
      },
      pollInterval: null,
      progressHistory: [],
      startTime: null,
      lastProgress: 0,
      lastUpdateTime: null
    }
  },
  computed: {
    progressPercent() {
      if (this.migrationState.total === 0) return 0
      return Math.round((this.migrationState.progress / this.migrationState.total) * 100)
    },
    migrationPhase() {
      if (!this.migrationState.message) return null

      // Check if reindexing
      if (this.migrationState.message.toLowerCase().includes('reindex')) {
        return 'Reindexing scenes'
      }

      // Check if migrating scenes
      if (this.migrationState.message.toLowerCase().includes('scene')) {
        return 'Migrating scene IDs'
      }

      return null
    },
    showETA() {
      // Show ETA for all progress including reindexing
      return this.migrationState.progress > 0
    },
    estimatedTimeRemaining() {
      if (!this.migrationState.total || !this.migrationState.progress || this.progressHistory.length < 3) {
        return null
      }

      const remaining = this.migrationState.total - this.migrationState.progress
      if (remaining <= 0) return null

      // Use all available history for more stable estimate
      const history = this.progressHistory
      if (history.length < 3) return null

      // Calculate rate from first to last point (overall average)
      const firstPoint = history[0]
      const lastPoint = history[history.length - 1]

      const totalItemsProcessed = lastPoint.progress - firstPoint.progress
      const totalTimeElapsed = (lastPoint.timestamp - firstPoint.timestamp) / 1000 // seconds

      if (totalItemsProcessed <= 0 || totalTimeElapsed <= 0) return null

      // Use overall average rate for more stable prediction
      const itemsPerSecond = totalItemsProcessed / totalTimeElapsed
      const secondsRemaining = remaining / itemsPerSecond

      // Apply smoothing: round to nearest 15 seconds to reduce jitter
      const smoothedSeconds = Math.round(secondsRemaining / 15) * 15

      return this.formatTime(smoothedSeconds)
    }
  },
  async mounted() {
    try {
      await this.checkMigrationStatus()
      // Only start polling if migrations are actually running
      if (this.isMigrating) {
        this.startPolling()
      }
    } catch (error) {
      console.error('Failed to check migration status:', error)
      // Don't block the UI if migration check fails
    }
  },
  beforeDestroy() {
    this.stopPolling()
  },
  methods: {
    async checkMigrationStatus() {
      try {
        const response = await ky.get('/api/options/state').json()
        if (response.currentState && response.currentState.migration) {
          const migration = response.currentState.migration
          this.migrationState = migration
          this.isMigrating = migration.is_running

          // Track progress history for ETA calculation
          if (migration.is_running && migration.progress > 0) {
            const now = Date.now()

            // Initialize start time on first progress
            if (!this.startTime) {
              this.startTime = now
            }

            // Record progress if it changed
            if (migration.progress !== this.lastProgress) {
              this.progressHistory.push({
                progress: migration.progress,
                timestamp: now
              })
              this.lastProgress = migration.progress
              this.lastUpdateTime = now

              // Keep more history for better averaging (last 20 updates)
              if (this.progressHistory.length > 20) {
                this.progressHistory.shift()
              }
            }
          }

          // Stop polling if migrations are complete
          if (!migration.is_running) {
            this.stopPolling()
          }
        }
      } catch (error) {
        console.error('Failed to check migration status:', error)
      }
    },
    formatTime(seconds) {
      if (seconds < 60) {
        return `${Math.round(seconds)}s`
      } else if (seconds < 3600) {
        const minutes = Math.floor(seconds / 60)
        const secs = Math.round(seconds % 60)
        return secs > 0 ? `${minutes}m ${secs}s` : `${minutes}m`
      } else {
        const hours = Math.floor(seconds / 3600)
        const minutes = Math.floor((seconds % 3600) / 60)
        return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
      }
    },
    startPolling() {
      this.pollInterval = setInterval(() => {
        this.checkMigrationStatus()
      }, 2000)
    },
    stopPolling() {
      if (this.pollInterval) {
        clearInterval(this.pollInterval)
        this.pollInterval = null
      }
    }
  }
}
</script>

<style scoped>
.modal-card {
  max-width: 600px;
}

.migration-progress {
  margin-top: 1.5rem;
}

.migration-phase {
  text-align: center;
  font-weight: 600;
  font-size: 1rem;
  color: #363636;
  margin-bottom: 0.75rem;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
  font-size: 0.95rem;
}

.progress-count {
  color: #7a7a7a;
  font-weight: 500;
}

.eta-text {
  color: #3273dc;
  font-weight: 600;
  font-size: 0.95rem;
}

.help-notice {
  font-size: 0.95rem;
  font-weight: 500;
  color: #3273dc;
}
</style>
