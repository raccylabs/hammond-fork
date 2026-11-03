<script setup>
import { FUEL_TYPES, FUEL_UNITS, distUnit } from '../lib/format.js'
defineProps({ model: { type: Object, required: true }, showCosts: { type: Boolean, default: true }, showOdo: Boolean })
</script>
<template>
  <div class="stack">
    <div class="grid2">
      <label class="field">Nickname<input v-model="model.nickname" placeholder="What you call it" /></label>
      <label class="field">Plate / registration<input v-model="model.registration" required /></label>
    </div>
    <div class="grid3">
      <label class="field">Year<input v-model.number="model.yearOfManufacture" inputmode="numeric" /></label>
      <label class="field">Make<input v-model="model.make" required /></label>
      <label class="field">Model<input v-model="model.model" required /></label>
    </div>
    <div class="grid3">
      <label class="field">Trim<input v-model="model.trim" /></label>
      <label class="field">Fuel
        <select v-model.number="model.fuelType"><option v-for="(l, k) in FUEL_TYPES" :key="k" :value="Number(k)">{{ l }}</option></select>
      </label>
      <label class="field">Measured in
        <select v-model.number="model.fuelUnit"><option v-for="(l, k) in FUEL_UNITS" :key="k" :value="Number(k)">{{ l }}</option></select>
      </label>
    </div>
    <div class="grid3">
      <label class="field">Drivetrain
        <select v-model="model.drivetrain"><option value="">Not sure</option><option>FWD</option><option>RWD</option><option>AWD</option><option>4WD</option></select>
      </label>
      <label class="field">Engine size (L)<input v-model.number="model.engineSize" inputmode="decimal" /></label>
      <label v-if="showOdo" class="field">Odometer ({{ distUnit() }})<input v-model.number="model.odometer" inputmode="numeric" /></label>
      <label v-else class="field">VIN<input v-model="model.vin" style="letter-spacing: 0.04em" /></label>
    </div>
    <div class="grid3">
      <label class="field">Oil<input v-model="model.oilType" placeholder="0W-20" /></label>
      <label class="field">Tire size<input v-model="model.tireSize" placeholder="225/50R17" /></label>
      <div class="grid2" style="gap: 8px">
        <label class="field">Front PSI<input v-model.number="model.tirePressureFront" inputmode="numeric" /></label>
        <label class="field">Rear PSI<input v-model.number="model.tirePressureRear" inputmode="numeric" /></label>
      </div>
    </div>
    <template v-if="showCosts">
      <div class="label" style="margin-top: 4px">For cost of ownership (optional)</div>
      <div class="grid2">
        <label class="field">Purchase date<input v-model="model.purchaseDay" type="date" /></label>
        <label class="field">Purchase price<input v-model.number="model.purchasePrice" inputmode="decimal" /></label>
      </div>
      <div class="grid2">
        <label class="field">Estimated value today<input v-model.number="model.estimatedValue" inputmode="decimal" /></label>
        <label class="field">Insurance per month<input v-model.number="model.insurancePremium" inputmode="decimal" /></label>
      </div>
    </template>
  </div>
</template>
