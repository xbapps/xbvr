<template>
  <div class="container">
    <b-loading :is-full-page="false" :active.sync="isLoading"></b-loading>
    <div class="content">
      <h3>{{$t("Task Schedules")}}</h3>
      <hr/>
      <b-tabs v-model="activeTab" size="medium" type="is-boxed" style="margin-left: 0px" id="importexporttab">
            <b-tab-item label="Rescrape"/>
            <b-tab-item label="Rescan"/>
            <b-tab-item label="Preview Generation"/>
            <b-tab-item label="Actor Rescrape"/>
            <b-tab-item label="Stashdb Rescrape"/>
            <b-tab-item :label="$t('Link Scenes')"/>
            <b-tab-item :label="$t('Auto Tags')"/>
      </b-tabs>
      <div class="columns">
        <div class="column">
          <section>
            <div v-if="activeTab == 0">
              <h4>{{$t("Scrape Sites")}}</h4>
              <b-field>
                <b-switch v-model="rescrapeEnabled">Enable schedule</b-switch>
              </b-field>
              <b-field v-if="rescrapeEnabled">
                <b-slider v-model="rescrapeHourInterval" :min="1" :max="23" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.rescrapeHourInterval} hour${this.rescrapeHourInterval > 1 ? 's': ''}`}}</div>
              </b-field>
              <b-field>
                <b-switch v-if="rescrapeEnabled" v-model="useRescrapeTimeRange">Limit time of day</b-switch>
              </b-field>
              <div v-if="useRescrapeTimeRange && rescrapeEnabled">
                <b-field>
                  <b-slider v-model="rescrapeTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictRescrapTo24Hours">
                    <b-slider-tick :value="0">00:00</b-slider-tick>
                    <b-slider-tick :value="6">06:00</b-slider-tick>
                    <b-slider-tick :value="12">12:00</b-slider-tick>
                    <b-slider-tick :value="18">18:00</b-slider-tick>
                    <b-slider-tick :value="24">Midnight</b-slider-tick>
                    <b-slider-tick :value="30">06:00</b-slider-tick>
                    <b-slider-tick :value="36">12:00</b-slider-tick>
                    <b-slider-tick :value="42">18:00</b-slider-tick>
                    <b-slider-tick :value="48">00:00</b-slider-tick>
                  </b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.rescrapeTimeRange[0]]} - ${this.timeRange[this.rescrapeTimeRange[1]]}`}}</div>
                </b-field>
                <b-field>
                  <b-slider v-model="rescrapeMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(rescrapeMinuteStart) }}</div>
                </b-field>
              </div>
              <br/>
              <b-field label="Startup">
                  <b-slider v-model="rescrapeStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(rescrapeStartDelay) }}</div>
              </b-field>
            </div>
            <div v-if="activeTab == 1">            
              <h4>{{$t("Rescan Folders")}}</h4>
              <b-field>
                <b-switch v-model="rescanEnabled">Enable schedule</b-switch>
              </b-field>
              <b-field v-if="rescanEnabled">
                <b-slider v-model="rescanHourInterval" :min="1" :max="23" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.rescanHourInterval} hour${this.rescanHourInterval > 1 ? 's': ''}`}}</div>
              </b-field>
              <b-field>
                <b-switch v-if="rescanEnabled" v-model="useRescanTimeRange">Limit time of day</b-switch>
              </b-field>
              <div v-if="useRescanTimeRange && rescanEnabled">
                <b-field>
                  <b-slider v-model="rescanTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictRescanTo24Hours">
                    <b-slider-tick :value="0">00:00</b-slider-tick>
                    <b-slider-tick :value="6">06:00</b-slider-tick>
                    <b-slider-tick :value="12">12:00</b-slider-tick>
                    <b-slider-tick :value="18">18:00</b-slider-tick>
                    <b-slider-tick :value="24">Midnight</b-slider-tick>
                    <b-slider-tick :value="30">06:00</b-slider-tick>
                    <b-slider-tick :value="36">12:00</b-slider-tick>
                    <b-slider-tick :value="42">18:00</b-slider-tick>
                    <b-slider-tick :value="48">00:00</b-slider-tick>
                  </b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.rescanTimeRange[0]]} - ${this.timeRange[this.rescanTimeRange[1]]}`}}</div>
                </b-field>
                <b-field>
                  <b-slider v-model="rescanMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(rescanMinuteStart) }}</div>
                </b-field>
              </div>
              <br/>
              <b-field label="Startup">
                  <b-slider v-model="rescanStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(rescanStartDelay) }}</div>
              </b-field>
            </div>
           <div v-if="activeTab == 2">            
              <b-field>
                <b-switch v-model="previewEnabled">Enable schedule</b-switch>
              </b-field>
              <b-field v-if="previewEnabled">
                <b-slider v-model="previewHourInterval" :min="1" :max="23" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.previewHourInterval} hour${this.previewHourInterval > 1 ? 's': ''}`}}</div>
              </b-field>
              <b-field>
                <b-switch v-if="previewEnabled" v-model="usePreviewTimeRange">Limit time of day</b-switch>
              </b-field>
              <div v-if="usePreviewTimeRange && previewEnabled">
                <b-field>
                  <b-slider v-model="previewTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictPreviewTo24Hours">
                    <b-slider-tick :value="0">00:00</b-slider-tick>
                    <b-slider-tick :value="6">06:00</b-slider-tick>
                    <b-slider-tick :value="12">12:00</b-slider-tick>
                    <b-slider-tick :value="18">18:00</b-slider-tick>
                    <b-slider-tick :value="24">Midnight</b-slider-tick>
                    <b-slider-tick :value="30">06:00</b-slider-tick>
                    <b-slider-tick :value="36">12:00</b-slider-tick>
                    <b-slider-tick :value="42">18:00</b-slider-tick>
                    <b-slider-tick :value="48">00:00</b-slider-tick>
                  </b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.previewTimeRange[0]]} - ${this.timeRange[this.previewTimeRange[1]]}`}}</div>
                </b-field>
                <b-field>
                  <b-slider v-model="previewMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(previewMinuteStart) }}</div>
                </b-field>
                <p>
                  Preview Generation of a scene will not start after the Time Window Ends
                </p>
              </div>
              <br/>
              <b-field label="Startup">
                  <b-slider v-model="previewStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(previewStartDelay) }}</div>
              </b-field>
              <p>
                BETA NOTE: Please note this is CPU-heavy process, if approriate limit the Time of Day the task runs                  
              </p>                  
            </div>
           <div v-if="activeTab == 3">            
              <b-field>
                <b-switch v-model="actorRescrapeEnabled">Enable schedule</b-switch>
              </b-field>
              <b-field v-if="actorRescrapeEnabled">
                <b-slider v-model="actorRescrapeHourInterval" :min="1" :max="23" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.actorRescrapeHourInterval} hour${this.actorRescrapeHourInterval > 1 ? 's': ''}`}}</div>
              </b-field>
              <b-field>
                <b-switch v-if="actorRescrapeEnabled" v-model="useActorRescrapeTimeRange">Limit time of day</b-switch>
              </b-field>
              <div v-if="useActorRescrapeTimeRange && actorRescrapeEnabled">
                <b-field>
                  <b-slider v-model="actorRescrapeTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictActorRescrapeTo24Hours">
                    <b-slider-tick :value="0">00:00</b-slider-tick>
                    <b-slider-tick :value="6">06:00</b-slider-tick>
                    <b-slider-tick :value="12">12:00</b-slider-tick>
                    <b-slider-tick :value="18">18:00</b-slider-tick>
                    <b-slider-tick :value="24">Midnight</b-slider-tick>
                    <b-slider-tick :value="30">06:00</b-slider-tick>
                    <b-slider-tick :value="36">12:00</b-slider-tick>
                    <b-slider-tick :value="42">18:00</b-slider-tick>
                    <b-slider-tick :value="48">00:00</b-slider-tick>
                  </b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.actorRescrapeTimeRange[0]]} - ${this.timeRange[this.actorRescrapeTimeRange[1]]}`}}</div>
                </b-field>
                <b-field>
                  <b-slider v-model="actorRescrapeMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(actorRescrapeMinuteStart) }}</div>
                </b-field>
              </div>
              <br/>
              <b-field label="Startup">
                  <b-slider v-model="actorRescrapeStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(actorRescrapeStartDelay) }}</div>
              </b-field>
            </div>
           <div v-if="activeTab == 4">            
              <b-field>
                <b-tooltip :active="stashApiKey==''" :label="$t('Enter a StashApi key to enable')" >
                  <b-switch v-model="stashdbRescrapeEnabled" :disabled="stashApiKey==''">Enable schedule</b-switch>
                </b-tooltip>
              </b-field>
              <b-field v-if="stashdbRescrapeEnabled">
                <b-slider v-model="stashdbRescrapeHourInterval" :min="1" :max="23" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.stashdbRescrapeHourInterval} hour${this.stashdbRescrapeHourInterval > 1 ? 's': ''}`}}</div>
              </b-field>
              <b-field>
                <b-switch v-if="stashdbRescrapeEnabled" v-model="useStashdbRescrapeTimeRange">Limit time of day</b-switch>
              </b-field>
              <div v-if="useStashdbRescrapeTimeRange && stashdbRescrapeEnabled">
                <b-field>
                  <b-slider v-model="stashdbRescrapeTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictStashdbRescrapeTo24Hours">
                    <b-slider-tick :value="0">00:00</b-slider-tick>
                    <b-slider-tick :value="6">06:00</b-slider-tick>
                    <b-slider-tick :value="12">12:00</b-slider-tick>
                    <b-slider-tick :value="18">18:00</b-slider-tick>
                    <b-slider-tick :value="24">Midnight</b-slider-tick>
                    <b-slider-tick :value="30">06:00</b-slider-tick>
                    <b-slider-tick :value="36">12:00</b-slider-tick>
                    <b-slider-tick :value="42">18:00</b-slider-tick>
                    <b-slider-tick :value="48">00:00</b-slider-tick>
                  </b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.stashdbRescrapeTimeRange[0]]} - ${this.timeRange[this.stashdbRescrapeTimeRange[1]]}`}}</div>
                </b-field>
                <b-field>
                  <b-slider v-model="stashdbRescrapeMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(stashdbRescrapeMinuteStart) }}</div>
                </b-field>
              </div>
              <br/>
              <b-field label="Startup">
                  <b-slider v-model="stashdbRescrapeStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                  <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(stashdbRescrapeStartDelay) }}</div>
              </b-field>
            </div>
           <div v-if="activeTab == 5">            
            <b-field>
              <b-switch v-model="linkScenesEnabled">Enable schedule</b-switch>
            </b-field>
            <b-field v-if="linkScenesEnabled">
              <b-slider v-model="linkScenesHourInterval" :min="1" :max="23" :step="1" ></b-slider>
              <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.linkScenesHourInterval} hour${this.linkScenesHourInterval > 1 ? 's': ''}`}}</div>
            </b-field>
            <b-field>
              <b-switch v-if="linkScenesEnabled" v-model="useLinkScenesTimeRange">Limit time of day</b-switch>
            </b-field>
            <div v-if="useLinkScenesTimeRange && linkScenesEnabled">
              <b-field>
                <b-slider v-model="linkScenesTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictLinkScenesTo24Hours">
                  <b-slider-tick :value="0">00:00</b-slider-tick>
                  <b-slider-tick :value="6">06:00</b-slider-tick>
                  <b-slider-tick :value="12">12:00</b-slider-tick>
                  <b-slider-tick :value="18">18:00</b-slider-tick>
                  <b-slider-tick :value="24">Midnight</b-slider-tick>
                  <b-slider-tick :value="30">06:00</b-slider-tick>
                  <b-slider-tick :value="36">12:00</b-slider-tick>
                  <b-slider-tick :value="42">18:00</b-slider-tick>
                  <b-slider-tick :value="48">00:00</b-slider-tick>
                </b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.linkScenesTimeRange[0]]} - ${this.timeRange[this.linkScenesTimeRange[1]]}`}}</div>
              </b-field>
              <b-field>
                <b-slider v-model="linkScenesMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(linkScenesMinuteStart) }}</div>
              </b-field>
              <p>
                Linking Scenes will not start after the Time Window Ends
              </p>
            </div>
            <br/>
            <b-field label="Startup">
                <b-slider v-model="linkScenesStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(linkScenesStartDelay) }}</div>
            </b-field>
          </div>
           <div v-if="activeTab == 6">
            <b-field>
              <b-switch v-model="autoTagScheduleEnabled">Enable schedule</b-switch>
            </b-field>
            <b-field v-if="autoTagScheduleEnabled">
              <b-slider v-model="autoTagScheduleHourInterval" :min="1" :max="23" :step="1" ></b-slider>
              <div class="column is-one-third" style="margin-left:.75em">{{`Run every ${this.autoTagScheduleHourInterval} hour${this.autoTagScheduleHourInterval > 1 ? 's': ''}`}}</div>
            </b-field>
            <b-field>
              <b-switch v-if="autoTagScheduleEnabled" v-model="useAutoTagScheduleTimeRange">Limit time of day</b-switch>
            </b-field>
            <div v-if="useAutoTagScheduleTimeRange && autoTagScheduleEnabled">
              <b-field>
                <b-slider v-model="autoTagScheduleTimeRange" :min="0" :max="48" :step="1" :custom-formatter="val => timeRange[val]" @input="restrictAutoTagScheduleTo24Hours">
                  <b-slider-tick :value="0">00:00</b-slider-tick>
                  <b-slider-tick :value="6">06:00</b-slider-tick>
                  <b-slider-tick :value="12">12:00</b-slider-tick>
                  <b-slider-tick :value="18">18:00</b-slider-tick>
                  <b-slider-tick :value="24">Midnight</b-slider-tick>
                  <b-slider-tick :value="30">06:00</b-slider-tick>
                  <b-slider-tick :value="36">12:00</b-slider-tick>
                  <b-slider-tick :value="42">18:00</b-slider-tick>
                  <b-slider-tick :value="48">00:00</b-slider-tick>
                </b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{`${this.timeRange[this.autoTagScheduleTimeRange[0]]} - ${this.timeRange[this.autoTagScheduleTimeRange[1]]}`}}</div>
              </b-field>
              <b-field>
                <b-slider v-model="autoTagScheduleMinuteStart" :min="0" :max="60" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{ minutesStartMsg(autoTagScheduleMinuteStart) }}</div>
              </b-field>
            </div>
            <br/>
            <b-field label="Startup">
                <b-slider v-model="autoTagScheduleStartDelay" :min="0" :max="60" :step="1" ></b-slider>
                <div class="column is-one-third" style="margin-left:.75em">{{ delayStartMsg(autoTagScheduleStartDelay) }}</div>
            </b-field>
            <hr/>
            <h4>{{$t("Generators")}}</h4>
            
            <div class="columns">
              <div class="column">
                <h5>{{$t("Actor Characteristics")}}</h5>
                <b-field>
                  <b-switch v-model="autoTagCupSize">
                    {{$t("Breast Size")}}
                    <br>
                    <small class="has-text-grey-light" v-if="breastSizeTags.length === 0">{{$t("Generates: Cup: C, Cup: DD, etc.")}}</small>
                    <b-taglist v-if="breastSizeTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in breastSizeTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagBreastType">
                    {{$t("Breast Type")}}
                    <br>
                    <small class="has-text-grey-light" v-if="breastTypeTags.length === 0">{{$t("Generates: Breast Type - Natural, Breast Type - Fake")}}</small>
                    <b-taglist v-if="breastTypeTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in breastTypeTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagAge">
                    {{$t("Age")}}
                    <br>
                    <small class="has-text-grey-light" v-if="ageTags.length === 0">{{$t("Generates: Age: 25, Age: 30, etc.")}}</small>
                    <b-taglist v-if="ageTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in ageTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagHeight">
                    {{$t("Height")}}
                    <br>
                    <small class="has-text-grey-light" v-if="heightTags.length === 0">{{$t("Generates: Height: Short/Average/Tall")}}</small>
                    <b-taglist v-if="heightTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in heightTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <div v-if="autoTagHeight" style="margin-left: 2em; margin-bottom: 1em;">
                  <b-field label="Short Max (cm)">
                    <b-input v-model="autoTagHeightShortMax" type="number"></b-input>
                  </b-field>
                  <b-field label="Average Max (cm)">
                    <b-input v-model="autoTagHeightAverageMax" type="number"></b-input>
                  </b-field>
                </div>

                <b-field>
                  <b-switch v-model="autoTagNationality">
                    {{$t("Nationality")}}
                    <br>
                    <small class="has-text-grey-light" v-if="nationalityTags.length === 0">{{$t("Generates: Nationality: USA, etc.")}}</small>
                    <b-taglist v-if="nationalityTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in nationalityTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagEthnicity">
                    {{$t("Ethnicity")}}
                    <br>
                    <small class="has-text-grey-light" v-if="ethnicityTags.length === 0">{{$t("Generates: Ethnicity: Asian, etc.")}}</small>
                    <b-taglist v-if="ethnicityTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in ethnicityTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagHairColor">
                    {{$t("Hair Color")}}
                    <br>
                    <small class="has-text-grey-light" v-if="hairColorTags.length === 0">{{$t("Generates: Hair: Blonde, etc.")}}</small>
                    <b-taglist v-if="hairColorTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in hairColorTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagEyeColor">
                    {{$t("Eye Color")}}
                    <br>
                    <small class="has-text-grey-light" v-if="eyeColorTags.length === 0">{{$t("Generates: Eyes: Blue, etc.")}}</small>
                    <b-taglist v-if="eyeColorTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in eyeColorTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
              </div>
              <div class="column">
                <h5>{{$t("Video Quality")}}</h5>
                <b-field>
                  <b-switch v-model="autoTagResolution">
                    {{$t("Resolution")}}
                    <br>
                    <small class="has-text-grey-light" v-if="resolutionTags.length === 0">{{$t("Generates: Res: 1080p, Res: 4K, etc.")}}</small>
                    <b-taglist v-if="resolutionTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in resolutionTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <b-field>
                  <b-switch v-model="autoTagVideoFormat">
                    {{$t("Video Format")}}
                    <br>
                    <small class="has-text-grey-light" v-if="videoFormatTags.length === 0">{{$t("Generates: Format: 180°, etc.")}}</small>
                    <b-taglist v-if="videoFormatTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in videoFormatTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>

                <h5 style="margin-top: 1em;">{{$t("Scene Attributes")}}</h5>
                <b-field>
                  <b-switch v-model="autoTagDuration">
                    {{$t("Duration")}}
                    <br>
                    <small class="has-text-grey-light" v-if="durationTags.length === 0">{{$t("Generates: Duration: Short/Standard/Long")}}</small>
                    <b-taglist v-if="durationTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in durationTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
                <div v-if="autoTagDuration" style="margin-left: 2em; margin-bottom: 1em;">
                  <b-field label="Short Max (min)">
                    <b-input v-model="autoTagDurationShortMax" type="number"></b-input>
                  </b-field>
                  <b-field label="Standard Max (min)">
                    <b-input v-model="autoTagDurationStandardMax" type="number"></b-input>
                  </b-field>
                </div>
                <b-field>
                  <b-switch v-model="autoTagInterracial">
                    {{$t("Interracial")}}
                    <br>
                    <small class="has-text-grey-light" v-if="interracialTags.length === 0">{{$t("Generates: Interracial")}}</small>
                    <b-taglist v-if="interracialTags.length > 0" style="margin-top: 5px;">
                      <b-tag v-for="tag in interracialTags" :key="tag.id" type="is-info" class="is-light" style="cursor: pointer; margin-right: 5px; margin-bottom: 2px;" @click.native.stop.prevent="showTagScenes(tag.name)">{{tag.name}} ({{tag.count}})</b-tag>
                    </b-taglist>
                  </b-switch>
                </b-field>
              </div>
            </div>

            <hr/>
            <h4>{{$t("Advanced Controls")}}</h4>
            <div class="columns">
               <div class="column">
                  <b-button type="is-info" icon-left="play" @click="runAutoTag" :loading="isRunNowLoading" style="margin-right: 1em">Run Now</b-button>
                  <b-button type="is-danger" icon-left="delete" @click="resetAutoTag" :loading="isResetLoading">Reset System Tags</b-button>
               </div>
            </div>
            


          </div>
            <hr/>
              <b-field grouped>
                <b-button type="is-primary" @click="saveSettings" style="margin-right:1em">Save settings</b-button>
              </b-field>
          </section>
          <hr/>
          <section>
            <p>
              Restart XBVR to use new schedule settings
            </p>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import prettyBytes from 'pretty-bytes'

export default {
  name: 'Schedules',
  data () {
    return {
      isLoading: true,
      activeTab: 0,
      rescrapeEnabled: true,
      rescanEnabled: true,
      rescrapeTimeRange: [0, 23],
      lastTimeRange: [0, 23],
      rescanTimeRange: [0, 23],
      lastrescanTimeRange: [0, 23],
      useRescrapeTimeRange: false,
      useRescanTimeRange: false,
      rescrapeHourInterval: 0,
      rescrapeMinuteStart: 0,
      rescanMinuteStart: 0,
      rescanHourInterval: 0,
      previewEnabled: false,
      previewTimeRange:[0,23],
      previewHourInterval: 0,
      previewMinuteStart: 0,
      lastPreviewTimeRange: [0,23],
      usePreviewTimeRange: false,      
      rescrapeStartDelay: 0,
      rescanStartDelay: 0,
      previewStartDelay: 0,
      actorRescrapeEnabled: false,
      actorRescrapeTimeRange:[0,23],
      actorRescrapeHourInterval: 0,
      actorRescrapeMinuteStart: 0,
      lastActorRescrapeTimeRange: [0,23],
      useActorRescrapeTimeRange: false,      
      actorRescrapeStartDelay: 0,
      stashdbRescrapeEnabled: false,
      stashdbRescrapeTimeRange:[0,23],
      stashdbRescrapeHourInterval: 0,
      stashdbRescrapeMinuteStart: 0,
      lastStashdbRescrapeTimeRange: [0,23],
      useStashdbRescrapeTimeRange: false,      
      stashdbRescrapeStartDelay: 0,
      linkScenesEnabled: false,
      linkScenesTimeRange:[0,23],
      linkScenesHourInterval: 0,
      linkScenesMinuteStart: 0,
      lastlinkScenesTimeRange: [0,23],
      useLinkScenesTimeRange: false,      
      linkScenesStartDelay: 0,
      autoTagScheduleEnabled: false,
      autoTagScheduleTimeRange:[0,23],
      autoTagScheduleHourInterval: 0,
      autoTagScheduleMinuteStart: 0,
      lastAutoTagScheduleTimeRange: [0,23],
      useAutoTagScheduleTimeRange: false,
      autoTagScheduleStartDelay: 0,
      autoTagBreastType: false,

      autoTagAge: false,
      autoTagHeight: false,
      autoTagNationality: false,
      autoTagEthnicity: false,
      autoTagHairColor: false,
      autoTagEyeColor: false,
      autoTagCupSize: false,
      autoTagResolution: false,
      autoTagVideoFormat: false,
      autoTagDuration: false,

      autoTagInterracial: false,
      autoTagHeightShortMax: 160,
      autoTagHeightAverageMax: 175,
      autoTagDurationShortMax: 15,
      autoTagDurationStandardMax: 40,
      isRunNowLoading: false,
      isResetLoading: false,
      systemTags: [],

      timeRange: ['00:00', '01:00', '02:00', '03:00', '04:00', '05:00', '06:00', '07:00', '08:00', '09:00', '10:00', '11:00',
        '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00', '19:00', '20:00', '21:00', '22:00', '23:00',
        '00:00', '01:00', '02:00', '03:00', '04:00', '05:00', '06:00', '07:00', '08:00', '09:00', '10:00', '11:00',
        '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00', '19:00', '20:00', '21:00', '22:00', '23:00', '00:00']
    }
  },
  async mounted () {
    await this.loadState()
    await this.loadSystemTags()
  },
  computed: {
    breastSizeTags () { return this.systemTags.filter(t => t.name.startsWith('Cup:')) },
    breastTypeTags () { return this.systemTags.filter(t => t.name.startsWith('Breast Type -')) },
    ageTags () { return this.systemTags.filter(t => t.name.startsWith('Age:')) },
    heightTags () { return this.systemTags.filter(t => t.name.startsWith('Height:')) },
    nationalityTags () { return this.systemTags.filter(t => t.name.startsWith('Nationality:')) },
    ethnicityTags () { return this.systemTags.filter(t => t.name.startsWith('Ethnicity:')) },
    hairColorTags () { return this.systemTags.filter(t => t.name.startsWith('Hair:')) },
    eyeColorTags () { return this.systemTags.filter(t => t.name.startsWith('Eyes:')) },
    resolutionTags () { return this.systemTags.filter(t => t.name.startsWith('Res:')) },
    videoFormatTags () { return this.systemTags.filter(t => t.name.startsWith('Format:')) },
    durationTags () { return this.systemTags.filter(t => t.name.startsWith('Duration:')) },
    interracialTags () { return this.systemTags.filter(t => t.name === 'Interracial') },

    stashApiKey: {
      get () {        
        return this.$store.state.optionsAdvanced.advanced.stashApiKey
      },
      set (value) {
        this.$store.state.optionsAdvanced.advanced.stashApiKey = value

      }
    },
  },
  methods: {
    restrictRescrapTo24Hours () {
      this.rescrapeTimeRange = this.restrictTo24Hours(this.rescrapeTimeRange, this.lastTimeRange)
      this.lastTimeRange = this.rescrapeTimeRange
    },
    restrictRescanTo24Hours () {
      this.rescanTimeRange = this.restrictTo24Hours(this.rescanTimeRange, this.lastrescanTimeRange)
      this.lastrescanTimeRange = this.rescanTimeRange
    },
    restrictPreviewTo24Hours () {
      this.previewTimeRange = this.restrictTo24Hours(this.previewTimeRange, this.lastPreviewTimeRange)
      this.lastPreviewTimeRange = this.previewTimeRange
    },
    restrictActorRescrapeTo24Hours () {
      this.actorRescrapeTimeRange = this.restrictTo24Hours(this.actorRescrapeTimeRange, this.lastActorRescrapeTimeRange)
      this.lastActorRescrapeTimeRange = this.actorRescrapeTimeRange
    },
    restrictStashdbRescrapeTo24Hours () {
      this.stashdbRescrapeTimeRange = this.restrictTo24Hours(this.stashdbRescrapeTimeRange, this.lastStashdbRescrapeTimeRange)
      this.lastStashdbRescrapeTimeRange = this.stashdbRescrapeTimeRange
    },
    restrictLinkScenesTo24Hours () {
      this.linkScenesTimeRange = this.restrictTo24Hours(this.linkScenesTimeRange, this.lastLinkScenesTimeRange)
      this.lastLinkScenesTimeRange = this.LinkScenesTimeRange
    },
    restrictAutoTagScheduleTo24Hours () {
      this.autoTagScheduleTimeRange = this.restrictTo24Hours(this.autoTagScheduleTimeRange, this.lastAutoTagScheduleTimeRange)
      this.lastAutoTagScheduleTimeRange = this.autoTagScheduleTimeRange
    },
    restrictTo24Hours (timeRange, lastTimeRange) {
      // check the first time is not in the second 24 hours, no need, should be in the first 24 hours
      if (timeRange[0] > 23) {
        timeRange[0] = 23
        timeRange = [timeRange[0], timeRange[1]]
      }
      // check they are not trying to select more than a 24 hour range
      if ((timeRange[1] - timeRange[0]) > 23 ) {
        if (timeRange[0] === lastTimeRange[0] || timeRange[0] === lastTimeRange[1]) {
          timeRange = [timeRange[1] - 23, timeRange[1]]
        } else {
          timeRange = [timeRange[0], timeRange[0] + 23]
        }
      }
      return timeRange
    },
    async runAutoTag () {
      this.isRunNowLoading = true
      await ky.get('/api/task/auto-tag')
      this.isRunNowLoading = false
      this.$buefy.toast.open({
        message: 'Auto-tagging started',
        type: 'is-success'
      })
      setTimeout(() => { this.loadSystemTags() }, 2000)
    },
    async loadSystemTags () {
      await ky.get('/api/task/system-tags')
        .json()
        .then(data => {
          this.systemTags = data
        })
    },
    showTagScenes (tagName) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = [tagName]
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
    },
    async resetAutoTag () {
      this.$buefy.dialog.confirm({
        title: 'Reset System Tags',
        message: 'Are you sure you want to delete all system-generated tags from all scenes?',
        confirmText: 'Reset',
        type: 'is-danger',
        hasIcon: true,
        onConfirm: async () => {
          this.isResetLoading = true
          await ky.get('/api/task/auto-tag-reset')
          this.isResetLoading = false
          this.$buefy.toast.open({
            message: 'System tags reset',
            type: 'is-success'
          })
          this.systemTags = []
        }
      })
    },

    async loadState () {
      this.isLoading = true
      await ky.get('/api/options/state')
        .json()
        .then(data => {
          this.rescrapeEnabled = data.config.cron.rescrapeSchedule.enabled
          this.rescrapeHourInterval = data.config.cron.rescrapeSchedule.hourInterval
          this.useRescrapeTimeRange = data.config.cron.rescrapeSchedule.useRange
          this.rescrapeMinuteStart = data.config.cron.rescrapeSchedule.minuteStart
          this.rescanEnabled = data.config.cron.rescanSchedule.enabled
          this.rescanHourInterval = data.config.cron.rescanSchedule.hourInterval
          this.useRescanTimeRange = data.config.cron.rescanSchedule.useRange
          this.rescanMinuteStart = data.config.cron.rescanSchedule.minuteStart
          this.previewEnabled = data.config.cron.previewSchedule.enabled
          this.previewHourInterval = data.config.cron.previewSchedule.hourInterval
          this.usePreviewTimeRange = data.config.cron.previewSchedule.useRange
          this.previewMinuteStart = data.config.cron.previewSchedule.minuteStart
          this.actorRescrapeEnabled = data.config.cron.actorRescrapeSchedule.enabled
          this.actorRescrapeHourInterval = data.config.cron.actorRescrapeSchedule.hourInterval
          this.useActorRescrapeTimeRange = data.config.cron.actorRescrapeSchedule.useRange
          this.actorRescrapeMinuteStart = data.config.cron.actorRescrapeSchedule.minuteStart          
          this.stashdbRescrapeEnabled = data.config.cron.stashdbRescrapeSchedule.enabled
          this.stashdbRescrapeHourInterval = data.config.cron.stashdbRescrapeSchedule.hourInterval
          this.useStashdbRescrapeTimeRange = data.config.cron.stashdbRescrapeSchedule.useRange
          this.stashdbRescrapeMinuteStart = data.config.cron.stashdbRescrapeSchedule.minuteStart          
          this.linkScenesEnabled = data.config.cron.linkScenesSchedule.enabled
          this.linkScenesHourInterval = data.config.cron.linkScenesSchedule.hourInterval
          this.useLinkScenesTimeRange = data.config.cron.linkScenesSchedule.useRange
          this.linkScenesMinuteStart = data.config.cron.linkScenesSchedule.minuteStart          
          if (data.config.cron.rescrapeSchedule.hourStart > data.config.cron.rescrapeSchedule.hourEnd) {
            this.rescrapeTimeRange = [data.config.cron.rescrapeSchedule.hourStart, data.config.cron.rescrapeSchedule.hourEnd + 24]
          } else {
            this.rescrapeTimeRange = [data.config.cron.rescrapeSchedule.hourStart, data.config.cron.rescrapeSchedule.hourEnd]
          }
          if (data.config.cron.rescanSchedule.hourStart > data.config.cron.rescanSchedule.hourEnd) {
            this.rescanTimeRange = [data.config.cron.rescanSchedule.hourStart, data.config.cron.rescanSchedule.hourEnd + 24]
          } else {
            this.rescanTimeRange = [data.config.cron.rescanSchedule.hourStart, data.config.cron.rescanSchedule.hourEnd]
          }
          if (data.config.cron.previewSchedule.hourStart > data.config.cron.previewSchedule.hourEnd) {
            this.previewTimeRange = [data.config.cron.previewSchedule.hourStart, data.config.cron.previewSchedule.hourEnd + 24]
          } else {
            this.previewTimeRange = [data.config.cron.previewSchedule.hourStart, data.config.cron.previewSchedule.hourEnd]            
          }
          if (data.config.cron.actorRescrapeSchedule.hourStart > data.config.cron.actorRescrapeSchedule.hourEnd) {
            this.actorRescrapeTimeRange = [data.config.cron.actorRescrapeSchedule.hourStart, data.config.cron.actorRescrapeSchedule.hourEnd + 24]
          } else {
            this.actorRescrapeTimeRange = [data.config.cron.actorRescrapeSchedule.hourStart, data.config.cron.actorRescrapeSchedule.hourEnd]            
          }
          
          if (data.config.cron.stashdbRescrapeSchedule.hourStart > data.config.cron.stashdbRescrapeSchedule.hourEnd) {
            this.stashdbRescrapeTimeRange = [data.config.cron.stashdbRescrapeSchedule.hourStart, data.config.cron.stashdbRescrapeSchedule.hourEnd + 24]
          } else {
            this.stashdbRescrapeTimeRange = [data.config.cron.stashdbRescrapeSchedule.hourStart, data.config.cron.stashdbRescrapeSchedule.hourEnd]            
          }

          if (data.config.cron.linkScenesSchedule.hourStart > data.config.cron.linkScenesSchedule.hourEnd) {
            this.linkScenesTimeRange = [data.config.cron.linkScenesSchedule.hourStart, data.config.cron.linkScenesSchedule.hourEnd + 24]
          } else {
            this.linkScenesTimeRange = [data.config.cron.linkScenesSchedule.hourStart, data.config.cron.linkScenesSchedule.hourEnd]            
          }

          this.autoTagScheduleEnabled = data.config.cron.autoTagSchedule.enabled
          this.autoTagScheduleHourInterval = data.config.cron.autoTagSchedule.hourInterval
          this.useAutoTagScheduleTimeRange = data.config.cron.autoTagSchedule.useRange
          this.autoTagScheduleMinuteStart = data.config.cron.autoTagSchedule.minuteStart

          if (data.config.cron.autoTagSchedule.hourStart > data.config.cron.autoTagSchedule.hourEnd) {
            this.autoTagScheduleTimeRange = [data.config.cron.autoTagSchedule.hourStart, data.config.cron.autoTagSchedule.hourEnd + 24]
          } else {
            this.autoTagScheduleTimeRange = [data.config.cron.autoTagSchedule.hourStart, data.config.cron.autoTagSchedule.hourEnd]
          }

          this.autoTagScheduleStartDelay = data.config.cron.autoTagSchedule.runAtStartDelay
          this.autoTagBreastType = data.config.autoTag.breastType

          this.autoTagAge = data.config.autoTag.age
          this.autoTagHeight = data.config.autoTag.height
          this.autoTagNationality = data.config.autoTag.nationality
          this.autoTagEthnicity = data.config.autoTag.ethnicity
          this.autoTagHairColor = data.config.autoTag.hairColor
          this.autoTagEyeColor = data.config.autoTag.eyeColor
          this.autoTagCupSize = data.config.autoTag.cupSize
          this.autoTagResolution = data.config.autoTag.resolution
          this.autoTagVideoFormat = data.config.autoTag.videoFormat
          this.autoTagDuration = data.config.autoTag.duration

          this.autoTagInterracial = data.config.autoTag.interracial
          this.autoTagHeightShortMax = data.config.autoTag.heightShortMax
          this.autoTagHeightAverageMax = data.config.autoTag.heightAverageMax
          this.autoTagDurationShortMax = data.config.autoTag.durationShortMax
          this.autoTagDurationStandardMax = data.config.autoTag.durationStandardMax
          
          this.rescrapeStartDelay = data.config.cron.rescrapeSchedule.runAtStartDelay
          this.rescanStartDelay = data.config.cron.rescanSchedule.runAtStartDelay          
          this.previewStartDelay = data.config.cron.previewSchedule.runAtStartDelay
          this.actorRescrapeStartDelay = data.config.cron.actorRescrapeSchedule.runAtStartDelay          
          this.stashdbRescrapeStartDelay = data.config.cron.stashdbRescrapeSchedule.runAtStartDelay          
          this.linkScenesStartDelay = data.config.cron.linkScenesSchedule.runAtStartDelay          
          this.isLoading = false
        })
    },
    minutesStartMsg (start) {
      if (start === 0) {
        return 'Start on the hour'
      }
      if (start === 1) {
        return 'Start at 1 minute past the hour'
      }
      return `Start at ${start} minutes past the hour`
    },
    delayStartMsg (start) {
      if (start === 0) {
        return 'Do not run at statup'
      }else{
        if (start === 1) {
          return `Run at 1 minute after startup`
        }else{
          return `Run at ${start} minutes after startup`
        }
      }
    },
    async saveSettings () {
      this.isLoading = true
      await ky.post('/api/options/task-schedule', {
        json: {
          rescrapeEnabled: this.rescrapeEnabled,
          rescrapeHourInterval: this.rescrapeHourInterval,
          rescrapeUseRange: this.useRescrapeTimeRange,
          rescrapeMinuteStart: this.rescrapeMinuteStart,
          rescrapeHourStart: this.rescrapeTimeRange[0],
          rescrapeHourEnd: this.rescrapeTimeRange[1],
          rescrapeStartDelay: this.rescrapeStartDelay,
          rescanEnabled: this.rescanEnabled,
          rescanHourInterval: this.rescanHourInterval,
          rescanUseRange: this.useRescanTimeRange,
          rescanMinuteStart: this.rescanMinuteStart,
          rescanHourStart: this.rescanTimeRange[0],
          rescanHourEnd: this.rescanTimeRange[1],
          rescanStartDelay: this.rescanStartDelay,
          previewEnabled: this.previewEnabled,
          previewHourInterval: this.previewHourInterval,
          previewUseRange: this.usePreviewTimeRange,
          previewMinuteStart: this.previewMinuteStart,
          previewHourStart: this.previewTimeRange[0],
          previewHourEnd: this.previewTimeRange[1],
          previewStartDelay:this.previewStartDelay,
          actorRescrapeEnabled: this.actorRescrapeEnabled,
          actorRescrapeHourInterval: this.actorRescrapeHourInterval,
          actorRescrapeUseRange: this.useActorRescrapeTimeRange,
          actorRescrapeMinuteStart: this.actorRescrapeMinuteStart,
          actorRescrapeHourStart: this.actorRescrapeTimeRange[0],
          actorRescrapeHourEnd: this.actorRescrapeTimeRange[1],
          actorRescrapeStartDelay:this.actorRescrapeStartDelay          ,
          stashdbRescrapeEnabled: this.stashdbRescrapeEnabled,
          stashdbRescrapeHourInterval: this.stashdbRescrapeHourInterval,
          stashdbRescrapeUseRange: this.useStashdbRescrapeTimeRange,
          stashdbRescrapeMinuteStart: this.stashdbRescrapeMinuteStart,
          stashdbRescrapeHourStart: this.stashdbRescrapeTimeRange[0],
          stashdbRescrapeHourEnd: this.stashdbRescrapeTimeRange[1],
          stashdbRescrapeStartDelay:this.stashdbRescrapeStartDelay,
          linkScenesEnabled: this.linkScenesEnabled,
          linkScenesHourInterval: this.linkScenesHourInterval,
          linkScenesUseRange: this.useLinkScenesTimeRange,
          linkScenesMinuteStart: this.linkScenesMinuteStart,
          linkScenesHourStart: this.linkScenesTimeRange[0],
          linkScenesHourEnd: this.linkScenesTimeRange[1],
          linkScenesStartDelay:this.linkScenesStartDelay,
          autoTagScheduleEnabled: this.autoTagScheduleEnabled,
          autoTagScheduleHourInterval: this.autoTagScheduleHourInterval,
          autoTagScheduleUseRange: this.useAutoTagScheduleTimeRange,
          autoTagScheduleMinuteStart: this.autoTagScheduleMinuteStart,
          autoTagScheduleHourStart: this.autoTagScheduleTimeRange[0],
          autoTagScheduleHourEnd: this.autoTagScheduleTimeRange[1],
          autoTagScheduleStartDelay:this.autoTagScheduleStartDelay,
          autoTagBreastType: this.autoTagBreastType,

          autoTagAge: this.autoTagAge,
          autoTagHeight: this.autoTagHeight,
          autoTagNationality: this.autoTagNationality,
          autoTagEthnicity: this.autoTagEthnicity,
          autoTagHairColor: this.autoTagHairColor,
          autoTagEyeColor: this.autoTagEyeColor,
          autoTagCupSize: this.autoTagCupSize,
          autoTagResolution: this.autoTagResolution,
          autoTagVideoFormat: this.autoTagVideoFormat,
          autoTagDuration: this.autoTagDuration,

          autoTagInterracial: this.autoTagInterracial,
          autoTagHeightShortMax: parseInt(this.autoTagHeightShortMax),
          autoTagHeightAverageMax: parseInt(this.autoTagHeightAverageMax),
          autoTagDurationShortMax: parseInt(this.autoTagDurationShortMax),
          autoTagDurationStandardMax: parseInt(this.autoTagDurationStandardMax)
        }
      })
        .json()
        .then(data => {
          this.isLoading = false
        })
    },
    prettyBytes
  }
}
</script>
