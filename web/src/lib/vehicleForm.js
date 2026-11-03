import { store } from './store.js'
import { isoDay, dayToISO } from './format.js'

export function blankVehicle() {
  const metric = store.user?.distanceUnit === 1
  return {
    nickname: '', registration: '', vin: '', make: '', model: '', yearOfManufacture: new Date().getFullYear(), trim: '',
    fuelType: 0, fuelUnit: metric ? 0 : 2, engineSize: 0, drivetrain: '', oilType: '', tireSize: '',
    tirePressureFront: 0, tirePressureRear: 0, purchaseDay: '', purchasePrice: 0, estimatedValue: 0, insurancePremium: 0, odometer: 0,
  }
}

export function fromVehicle(v) {
  return { ...blankVehicle(), ...v, purchaseDay: v.purchaseDate && !String(v.purchaseDate).startsWith('0001') ? isoDay(v.purchaseDate) : '' }
}

export function toPayload(m) {
  const n = (x) => (x === '' || x == null || isNaN(x) ? 0 : Number(x))
  return {
    nickname: m.nickname || `${m.make} ${m.model}`.trim(), registration: m.registration || '—', vin: (m.vin || '').toUpperCase(),
    make: m.make, model: m.model, yearOfManufacture: n(m.yearOfManufacture), engineSize: n(m.engineSize),
    fuelUnit: n(m.fuelUnit), fuelType: n(m.fuelType), trim: m.trim, drivetrain: m.drivetrain, oilType: m.oilType, tireSize: m.tireSize,
    tirePressureFront: n(m.tirePressureFront), tirePressureRear: n(m.tirePressureRear),
    purchaseDate: m.purchaseDay ? dayToISO(m.purchaseDay) : null, purchasePrice: n(m.purchasePrice), estimatedValue: n(m.estimatedValue),
    insurancePremium: n(m.insurancePremium), odometer: n(m.odometer),
  }
}
