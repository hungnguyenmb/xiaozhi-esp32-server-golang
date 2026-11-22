import vi from 'element-plus/es/locale/lang/vi'

// Complete accessibility labels missing from Element Plus's bundled Vietnamese locale.
export default {
  ...vi,
  el: {
    ...vi.el,
    breadcrumb: { label: 'Đường dẫn điều hướng' },
    colorpicker: { ...vi.el.colorpicker, defaultLabel: 'Chọn màu', description: 'Màu hiện tại là {color}. Nhấn Enter để chọn màu khác.', alphaLabel: 'Chọn độ trong suốt' },
    datepicker: {
      ...vi.el.datepicker,
      dateTablePrompt: 'Dùng phím mũi tên và Enter để chọn ngày',
      monthTablePrompt: 'Dùng phím mũi tên và Enter để chọn tháng',
      yearTablePrompt: 'Dùng phím mũi tên và Enter để chọn năm',
      selectedDate: 'Ngày đã chọn', week: 'Tuần',
      weeksFull: { sun: 'Chủ nhật', mon: 'Thứ hai', tue: 'Thứ ba', wed: 'Thứ tư', thu: 'Thứ năm', fri: 'Thứ sáu', sat: 'Thứ bảy' }
    },
    inputNumber: { decrease: 'Giảm giá trị', increase: 'Tăng giá trị' },
    dropdown: { toggleDropdown: 'Mở hoặc đóng danh sách' },
    pagination: { ...vi.el.pagination, deprecationWarning: 'Cách sử dụng này đã cũ. Vui lòng xem tài liệu el-pagination.' },
    dialog: { close: 'Đóng hộp thoại' },
    drawer: { close: 'Đóng bảng điều khiển' },
    messagebox: { ...vi.el.messagebox, close: 'Đóng hộp thoại' },
    slider: { defaultLabel: 'Thanh trượt từ {min} đến {max}', defaultRangeStartLabel: 'Chọn giá trị đầu', defaultRangeEndLabel: 'Chọn giá trị cuối' }
  }
}
